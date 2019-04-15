// Responsible for integrity of your data by periodically scanning your volumes to detect
// bit rot and hardware failures.
package varastointegrityverifier

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"go.etcd.io/bbolt"
	"io"
	"io/ioutil"
	"log"
	"time"
)

const errorReportMaxLength = 20 * 1024

type Controller struct {
	db                  *bolt.DB
	runningJobIds       map[string]*stopper.Stopper
	driverByVolumeId    map[int]blobdriver.Driver
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
	driverByVolumeId map[int]blobdriver.Driver,
	logger *log.Logger,
	stop *stopper.Stopper,
) *Controller {
	ctrl := &Controller{
		db:                  db,
		ivJobRepository:     ivJobRepository,
		blobRepository:      blobRepository,
		runningJobIds:       map[string]*stopper.Stopper{},
		driverByVolumeId:    driverByVolumeId,
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

				for id, _ := range ctrl.runningJobIds {
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

	volumeDriver, exists := s.driverByVolumeId[job.VolumeId]
	if !exists {
		return errors.New("volume not found")
	}

	s.runningJobIds[jobId] = stop

	go func() {
		defer stop.Done()

		if err := s.resumeJobWorker(job, volumeDriver, stop); err != nil {
			s.logl.Error.Printf("resumeJobWorker: %v", err)
		}

		delete(s.runningJobIds, jobId)
	}()

	return nil
}

func (s *Controller) nextBlobsForJob(lastCompletedBlobRef varastotypes.BlobRef, limit int) ([]varastotypes.Blob, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blobs := []varastotypes.Blob{}

	return blobs, s.blobRepository.EachFrom([]byte(lastCompletedBlobRef), func(record interface{}) error {
		blobs = append(blobs, *record.(*varastotypes.Blob))

		if len(blobs) >= limit {
			return blorm.StopIteration
		}

		return nil
	}, tx)
}

func (s *Controller) resumeJobWorker(
	job *varastotypes.IntegrityVerificationJob,
	volumeDriver blobdriver.Driver,
	stop *stopper.Stopper,
) error {
	lastStatusUpdate := time.Now()

	updateJobStatusInDb := func() error {
		return s.db.Update(func(tx *bolt.Tx) error {
			return s.ivJobRepository.Update(job, tx)
		})
	}
	defer updateJobStatusInDb() // to cover all following returns. ignores error

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

			bytesScanned, err := verifyOneBlob(blob.Ref, volumeDriver)
			if err != nil {
				job.ErrorsFound++
				job.Report += fmt.Sprintf("blob %s: %v\n", blob.Ref.AsHex(), err)

				if len(job.Report) > errorReportMaxLength {
					job.Report += "maximum errors detected; aborting job"
					return errors.New("maximum errors detected")
				}
			}

			job.BytesScanned += uint64(bytesScanned)
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

func (s *Controller) loadJob(jobId string) (*varastotypes.IntegrityVerificationJob, error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	job := &varastotypes.IntegrityVerificationJob{}
	if err := s.ivJobRepository.OpenByPrimaryKey([]byte(jobId), job, tx); err != nil {
		return nil, err
	}

	return job, nil
}

func verifyOneBlob(ref varastotypes.BlobRef, volumeDriver blobdriver.Driver) (int64, error) {
	blobContent, err := volumeDriver.Fetch(ref)
	if err != nil {
		return 0, err
	}

	blobVerifiedContent := varastoutils.BlobHashVerifier(blobContent, ref)

	// even though we ignore content, BlobHashVerifier middleware will yield us error
	// if hash is not correct
	bytesScanned, err := io.Copy(ioutil.Discard, blobVerifiedContent)
	if err != nil {
		return bytesScanned, err
	}

	return bytesScanned, nil
}
