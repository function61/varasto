// Responsible for integrity of your data by periodically scanning your volumes to detect
// bit rot and hardware failures.
package stointegrityverifier

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"log"
	"time"
)

const errorReportMaxLength = 20 * 1024

type Controller struct {
	db                  *bolt.DB
	runningJobIds       map[string]*stopper.Stopper
	diskAccess          *stodiskaccess.Controller
	ivJobRepository     blorm.Repository
	blobRepository      blorm.Repository
	resume              chan string
	stop                chan string
	opListRunningJobIds chan chan []string
	logl                *logex.Leveled
}

func (s *Controller) Resume(jobId string) {
	s.resume <- jobId
}

func (s *Controller) Stop(jobId string) {
	s.stop <- jobId
}

func (s *Controller) ListRunningJobs() []string {
	op := make(chan []string, 1)
	s.opListRunningJobIds <- op
	return <-op
}

// returns controller with threadsafe APIs whose work will be safely executed in a single thread
func NewController(
	db *bolt.DB,
	ivJobRepository blorm.Repository,
	blobRepository blorm.Repository,
	diskAccess *stodiskaccess.Controller,
	logger *log.Logger,
	stop *stopper.Stopper,
) *Controller {
	ctrl := &Controller{
		db:                  db,
		ivJobRepository:     ivJobRepository,
		blobRepository:      blobRepository,
		runningJobIds:       map[string]*stopper.Stopper{},
		diskAccess:          diskAccess,
		resume:              make(chan string, 1),
		stop:                make(chan string, 1),
		opListRunningJobIds: make(chan chan []string),
		logl:                logex.Levels(logger),
	}

	go func() {
		defer stop.Done()
		defer ctrl.logl.Info.Println("stopped")

		subWorkers := stopper.NewManager()

		for {
			select {
			case <-stop.Signal:
				subWorkers.StopAllWorkersAndWait()
				return
			case jobId := <-ctrl.stop:
				stop, found := ctrl.runningJobIds[jobId]
				if !found {
					ctrl.logl.Error.Printf("did not find job %s", jobId)
					continue
				}

				ctrl.logl.Info.Printf("stopping job %s", jobId)
				stop.SignalStop()
			case jobId := <-ctrl.resume:
				ctrl.logl.Info.Printf("resuming job %s", jobId)

				if err := ctrl.resumeJob(jobId, db, subWorkers.Stopper()); err != nil {
					ctrl.logl.Error.Printf("resumeJob: %v", err)
				}
			case result := <-ctrl.opListRunningJobIds:
				jobIds := []string{}

				for id := range ctrl.runningJobIds {
					jobIds = append(jobIds, id)
				}

				result <- jobIds
			}
		}
	}()

	return ctrl
}

func (s *Controller) resumeJob(jobId string, db *bolt.DB, stop *stopper.Stopper) error {
	if _, running := s.runningJobIds[jobId]; running {
		return errors.New("job is already running")
	}
	job, err := s.loadJob(jobId)
	if err != nil {
		return err
	}

	s.runningJobIds[jobId] = stop

	go func() {
		defer stop.Done()

		if err := s.resumeJobWorker(job, stop); err != nil {
			s.logl.Error.Printf("resumeJobWorker: %v", err)
		}

		delete(s.runningJobIds, jobId)
	}()

	return nil
}

func (s *Controller) nextBlobsForJob(lastCompletedBlobRef stotypes.BlobRef, limit int) ([]stotypes.Blob, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer ignoreError(tx.Rollback())

	blobs := []stotypes.Blob{}

	return blobs, s.blobRepository.EachFrom([]byte(lastCompletedBlobRef), func(record interface{}) error {
		blobs = append(blobs, *record.(*stotypes.Blob))

		if len(blobs) >= limit {
			return blorm.StopIteration
		}

		return nil
	}, tx)
}

func (s *Controller) resumeJobWorker(
	job *stotypes.IntegrityVerificationJob,
	stop *stopper.Stopper,
) error {
	lastStatusUpdate := time.Now()

	updateJobStatusInDb := func() error {
		return s.db.Update(func(tx *bolt.Tx) error {
			return s.ivJobRepository.Update(job, tx)
		})
	}
	defer ignoreError(updateJobStatusInDb()) // to cover all following returns. ignores error

	// returns error if maximum errors detected and the job should stop
	pushErr := func(reportLine string) error {
		job.ErrorsFound++
		job.Report += reportLine

		if len(job.Report) > errorReportMaxLength {
			job.Report += "maximum errors detected; aborting job"
			return errors.New("maximum errors detected")
		}

		return nil
	}

	batchLimit := 1000

	for {
		// discover next batch
		// FIXME: this always fetches the last blob of previous batch to the next batch
		blobBatch, err := s.nextBlobsForJob(job.LastCompletedBlobRef, batchLimit)
		if err != nil {
			return err
		}

		if len(blobBatch) == 0 { // completed
			break
		}

		// verify them
		for _, blob := range blobBatch {
			blobExistsOnVolumeToVerify := sliceutil.ContainsInt(blob.Volumes, job.VolumeId)
			if !blobExistsOnVolumeToVerify {
				continue
			}

			s.logl.Debug.Printf("verifying %s", blob.Ref.AsHex())

			bytesScanned, err := s.diskAccess.Scrub(blob.Ref, job.VolumeId)
			if err != nil {
				descr := fmt.Sprintf("blob %s: %v\n", blob.Ref.AsHex(), err)
				if err := pushErr(descr); err != nil {
					return err
				}
			}
			if int32(bytesScanned) != blob.SizeOnDisk {
				descr := fmt.Sprintf("blob %s size mismatch; expected=%d got=%d\n", blob.Ref.AsHex(), blob.SizeOnDisk, bytesScanned)
				if err := pushErr(descr); err != nil {
					return err
				}
			}

			job.BytesScanned += uint64(blob.SizeOnDisk)
			job.LastCompletedBlobRef = blob.Ref

			select {
			case <-stop.Signal:
				return nil
			default:
			}

			if time.Since(lastStatusUpdate) >= 5*time.Second {
				if err := updateJobStatusInDb(); err != nil {
					return err
				}

				lastStatusUpdate = time.Now()
			}
		}

		if len(blobBatch) < batchLimit { // fewer blobs than requested, so there will be no more
			break
		}
	}

	job.Completed = time.Now()
	job.Report += fmt.Sprintf("Completed with %d error(s)\n", job.ErrorsFound)

	s.logl.Debug.Println("finished")

	return nil
}

func (s *Controller) loadJob(jobId string) (*stotypes.IntegrityVerificationJob, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer ignoreError(tx.Rollback())

	job := &stotypes.IntegrityVerificationJob{}
	if err := s.ivJobRepository.OpenByPrimaryKey([]byte(jobId), job, tx); err != nil {
		return nil, err
	}

	return job, nil
}

func ignoreError(err error) {
	// no-op
}
