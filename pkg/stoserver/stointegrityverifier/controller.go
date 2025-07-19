// Responsible for integrity of your data by periodically scanning your volumes to detect
// bit rot and hardware failures.
package stointegrityverifier

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

const errorReportMaxLength = 20 * 1024

type Controller struct {
	db                  *bbolt.DB
	runningJobIDs       map[string]context.CancelFunc
	diskAccess          *stodiskaccess.Controller
	ivJobRepository     blorm.Repository
	blobRepository      blorm.Repository
	resume              chan string
	stop                chan string
	stopped             chan string
	opListRunningJobIDs chan chan []string
	logl                *logex.Leveled
}

// public API

func (c *Controller) Resume(jobID string) {
	c.resume <- jobID
}

func (c *Controller) Stop(jobID string) {
	c.stop <- jobID
}

func (c *Controller) ListRunningJobs() []string {
	op := make(chan []string, 1)
	c.opListRunningJobIDs <- op
	return <-op
}

// returns controller with threadsafe APIs whose work will be safely executed in a single thread
func NewController(
	db *bbolt.DB,
	ivJobRepository blorm.Repository,
	blobRepository blorm.Repository,
	diskAccess *stodiskaccess.Controller,
	logger *log.Logger,
	start func(fn func(context.Context) error),
) *Controller {
	ctrl := &Controller{
		db:                  db,
		ivJobRepository:     ivJobRepository,
		blobRepository:      blobRepository,
		runningJobIDs:       map[string]context.CancelFunc{},
		diskAccess:          diskAccess,
		resume:              make(chan string, 1),
		stop:                make(chan string, 1),
		stopped:             make(chan string, 1),
		opListRunningJobIDs: make(chan chan []string),
		logl:                logex.Levels(logger),
	}

	start(func(ctx context.Context) error {
		return ctrl.run(ctx)
	})

	return ctrl
}

func (c *Controller) run(ctx context.Context) error {
	handleStopped := func(jobID string) {
		delete(c.runningJobIDs, jobID)
	}

	for {
		select {
		case <-ctx.Done():
			// wait for all to stop
			for len(c.runningJobIDs) > 0 {
				c.logl.Info.Printf("waiting %d job(s) to stop", len(c.runningJobIDs))

				handleStopped(<-c.stopped)
			}

			return nil
		case jobID := <-c.stop:
			jobCancel, found := c.runningJobIDs[jobID]
			if !found {
				c.logl.Error.Printf("did not find job %s", jobID)
				continue
			}

			c.logl.Info.Printf("stopping job %s", jobID)
			jobCancel()
		case jobID := <-c.stopped:
			handleStopped(jobID)
		case jobID := <-c.resume:
			c.logl.Info.Printf("resuming job %s", jobID)

			if err := c.resumeJob(ctx, jobID); err != nil {
				c.logl.Error.Printf("resumeJob: %v", err)
			}
		case result := <-c.opListRunningJobIDs:
			jobIds := []string{}

			for id := range c.runningJobIDs {
				jobIds = append(jobIds, id)
			}

			result <- jobIds
		}
	}
}

func (c *Controller) resumeJob(ctx context.Context, jobID string) error {
	if _, running := c.runningJobIDs[jobID]; running {
		return errors.New("job is already running")
	}
	job, err := c.loadJob(jobID)
	if err != nil {
		return err
	}

	// job cancellation:
	// a) *all jobs* on parent cancel (program stopping) OR
	// b) individual job cancel via public API Stop()
	jobCtx, cancel := context.WithCancel(ctx)

	c.runningJobIDs[jobID] = cancel

	go func() {
		defer cancel()

		if err := c.resumeJobWorker(jobCtx, job); err != nil {
			c.logl.Error.Printf("resumeJobWorker: %v", err)
		}

		c.stopped <- jobID
	}()

	return nil
}

func (c *Controller) nextBlobsForJob(lastCompletedBlobRef stotypes.BlobRef, limit int) ([]stotypes.Blob, error) {
	tx, err := c.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	blobs := []stotypes.Blob{}

	return blobs, c.blobRepository.EachFrom([]byte(lastCompletedBlobRef), func(record any) error {
		blobs = append(blobs, *record.(*stotypes.Blob))

		if len(blobs) >= limit {
			return blorm.ErrStopIteration
		}

		return nil
	}, tx)
}

func (c *Controller) resumeJobWorker(
	ctx context.Context,
	job *stotypes.IntegrityVerificationJob,
) error {
	lastStatusUpdate := time.Now()

	updateJobStatusInDB := func() error {
		return c.db.Update(func(tx *bbolt.Tx) error {
			return c.ivJobRepository.Update(job, tx)
		})
	}
	defer func() { ignoreError(updateJobStatusInDB()) }() // to cover all following returns. ignores error

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
		blobBatch, err := c.nextBlobsForJob(job.LastCompletedBlobRef, batchLimit)
		if err != nil {
			return err
		}

		if len(blobBatch) == 0 { // completed
			break
		}

		// verify them
		for _, blob := range blobBatch {
			// not strictly completed (as we just begun work on it), but if we have lots of
			// blobs overall, and this exact volume has very few, and we'd skip updating this
			// after "blobExistsOnVolumeToVerify" check, we'd receive very little status updates
			job.LastCompletedBlobRef = blob.Ref

			if time.Since(lastStatusUpdate) >= 5*time.Second {
				if err := updateJobStatusInDB(); err != nil {
					return err
				}

				lastStatusUpdate = time.Now()

				select {
				case <-ctx.Done():
					return nil
				default:
				}
			}

			blobExistsOnVolumeToVerify := slices.Contains(blob.Volumes, job.VolumeID)
			if !blobExistsOnVolumeToVerify {
				continue
			}

			bytesScanned, err := c.diskAccess.Scrub(blob.Ref, job.VolumeID)
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
		}

		if len(blobBatch) < batchLimit { // fewer blobs than requested, so there will be no more
			break
		}
	}

	job.Completed = time.Now()
	job.Report += fmt.Sprintf("Completed with %d error(s)\n", job.ErrorsFound)

	c.logl.Debug.Println("finished")

	return nil
}

func (c *Controller) loadJob(jobID string) (*stotypes.IntegrityVerificationJob, error) {
	tx, err := c.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	job := &stotypes.IntegrityVerificationJob{}
	if err := c.ivJobRepository.OpenByPrimaryKey([]byte(jobID), job, tx); err != nil {
		return nil, err
	}

	return job, nil
}

func ignoreError(err error) {
	// no-op
}
