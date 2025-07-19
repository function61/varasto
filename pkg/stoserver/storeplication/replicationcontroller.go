// Controls replication of data between volumes
package storeplication

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

type replicationJob struct {
	Ref          stotypes.BlobRef
	FromVolumeID int
}

type replicationStats struct {
	blobVolumeNotAccessible int64
	otherErrors             int64
}

type Controller struct {
	toVolumeID int
	progress   *atomicInt32
	logl       *logex.Leveled
	db         *bbolt.DB
	diskAccess *stodiskaccess.Controller
	stats      replicationStats
}

// returns controller API and a function you must call (maybe in a separate goroutine) to run the logic
func New(
	toVolumeID int,
	db *bbolt.DB,
	diskAccess *stodiskaccess.Controller,
	logger *log.Logger,
	start func(fn func(context.Context) error),
) *Controller {
	c := &Controller{
		toVolumeID: toVolumeID,
		progress:   newAtomicInt32(0),
		logl:       logex.Levels(logger),
		db:         db,
		diskAccess: diskAccess,
	}

	start(func(ctx context.Context) error { return c.run(ctx) })

	return c
}

func (c *Controller) Progress() int {
	return int(c.progress.Get())
}

// reports if the controller is having any errors
func (c *Controller) Error() error {
	if c.stats.blobVolumeNotAccessible > 0 || c.stats.otherErrors > 0 {
		return fmt.Errorf("blobVolumeNotAccessible=%d otherErrors=%d", c.stats.blobVolumeNotAccessible, c.stats.otherErrors)
	}

	return nil
}

func (c *Controller) run(ctx context.Context) error {
	continueToken := stodb.StartFromFirst

	fiveSeconds := time.NewTicker(5 * time.Second)

	for {
		// give priority to stop signal
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		select {
		case <-ctx.Done():
			return nil
		case <-fiveSeconds.C:
			atBeginning := bytes.Equal(continueToken, stodb.StartFromFirst)
			if atBeginning { // probably looped to beginning from previously passing it fully ..
				// .. hence start stats from over
				c.stats = replicationStats{}
			}

			nextContinueToken, err := c.discoverAndRunReplicationJobs(ctx, continueToken)
			if err != nil {
				c.logl.Error.Printf("discoverAndRunReplicationJobs: %v", err)
				time.Sleep(3 * time.Second) // to not bombard with errors at full speed
			}

			continueToken = nextContinueToken
		}
	}
}

func (c *Controller) discoverAndRunReplicationJobs(
	ctx context.Context,
	continueToken []byte,
) ([]byte, error) {
	jobs, nextContinueToken, err := c.discoverReplicationJobs(continueToken)
	if err != nil {
		return nextContinueToken, err
	}

	// cap is the amount of runners we'll spawn
	// on lower end hardware the replication can be CPU bound, so let's use concurrency
	// n=3 => 43.19 GB/h
	// n=6 => 47.26 GB/h
	jobQueue := make(chan *replicationJob, 4)

	jobRunnersDone := sync.WaitGroup{}

	runner := func() {
		defer jobRunnersDone.Done()

		for job := range jobQueue {
			if err := c.replicateJob(job); err != nil {
				c.logl.Error.Printf("replicating blob %s: %v", job.Ref.AsHex(), err)
				c.stats.otherErrors++
				time.Sleep(1 * time.Second) // to not bombard with errors at full speed
			}
		}
	}

	for i := 0; i < cap(jobQueue); i++ {
		jobRunnersDone.Add(1)
		go runner()
	}

	defer func() {
		close(jobQueue)

		jobRunnersDone.Wait()
	}()

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return nextContinueToken, nil
		default:
		}

		jobQueue <- job

		// our database is btree so iteration of blobs is sorted, and keys are hashes
		// with random distribution, so we can estimate progress by looking at first
		// 16 bits of hash
		c.progress.Set(int32(float64(binary.BigEndian.Uint16(job.Ref[0:2])) / 65536.0 * 100.0))
	}

	// if we didn't find any jobs there'd be no `progress.Set()` call from the above loop
	if bytes.Equal(nextContinueToken, stodb.StartFromFirst) {
		c.progress.Set(100)
	}

	return nextContinueToken, nil
}

func (c *Controller) replicateJob(job *replicationJob) error {
	// intentionally using background context as not to unnecessarily cancel write to a
	// blobstore driver - one blob write is expected to take so little time we can wait the
	// pending ones out).
	return c.diskAccess.Replicate(
		context.Background(),
		job.FromVolumeID,
		c.toVolumeID,
		job.Ref)
}

func (c *Controller) discoverReplicationJobs(continueToken []byte) ([]*replicationJob, []byte, error) {
	tx, err := c.db.Begin(false)
	if err != nil {
		return nil, continueToken, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	batchLimit := 500

	jobs := []*replicationJob{}

	nextContinueToken := stodb.StartFromFirst

	err = stodb.BlobsPendingReplicationByVolumeIndex.Query(volIDToBytesForIndex(c.toVolumeID), continueToken, func(id []byte) error {
		if len(jobs) == batchLimit {
			nextContinueToken = id

			c.logl.Info.Printf(
				"operating @ batchLimit (%d)",
				batchLimit)
			return stodb.StopIteration
		}

		ref := stotypes.BlobRef(id)

		blob, err := stodb.Read(tx).Blob(ref)
		if err != nil {
			return err
		}

		if !sliceutil.ContainsInt(blob.VolumesPendingReplication, c.toVolumeID) {
			return fmt.Errorf(
				"blob %s volume %d not pending replication (but found from index query)",
				ref.AsHex(),
				c.toVolumeID)
		}

		bestFromVolume, err := c.diskAccess.BestVolumeID(blob.Volumes)
		if err != nil {
			if err == stotypes.ErrBlobNotAccessibleOnThisNode {
				c.stats.blobVolumeNotAccessible++
				return nil
			} else {
				c.stats.otherErrors++
				return err
			}
		}

		jobs = append(jobs, &replicationJob{
			Ref:          blob.Ref,
			FromVolumeID: bestFromVolume,
		})

		return nil
	}, tx)

	return jobs, nextContinueToken, err
}

func HasQueuedWriteIOsForVolume(volID int, tx *bbolt.Tx) (bool, error) {
	anyQueued := false
	if err := stodb.BlobsPendingReplicationByVolumeIndex.Query(
		volIDToBytesForIndex(volID),
		stodb.StartFromFirst,
		func(_ []byte) error {
			anyQueued = true
			return stodb.StopIteration
		},
		tx,
	); err != nil {
		return false, err
	}

	return anyQueued, nil
}

func volIDToBytesForIndex(volID int) []byte {
	return []byte(fmt.Sprintf("%d", volID))
}

type atomicInt32 struct {
	num *int32
}

func newAtomicInt32(initialValue int32) *atomicInt32 {
	return &atomicInt32{
		num: &initialValue,
	}
}

func (a *atomicInt32) Get() int32 {
	return atomic.LoadInt32(a.num)
}

func (a *atomicInt32) Set(val int32) {
	atomic.StoreInt32(a.num, val)
}

func ignoreError(err error) {
	// no-op
}
