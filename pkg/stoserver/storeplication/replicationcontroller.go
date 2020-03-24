// Controls replication of data between volumes
package storeplication

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type replicationJob struct {
	Ref          stotypes.BlobRef
	FromVolumeId int
}

type Controller struct {
	toVolumeId int
	progress   *atomicInt32
	stop       *stopper.Stopper
	logl       *logex.Leveled
	db         *bbolt.DB
	diskAccess *stodiskaccess.Controller
}

func Start(toVolumeId int, db *bbolt.DB, diskAccess *stodiskaccess.Controller, logger *log.Logger, stop *stopper.Stopper) *Controller {
	c := &Controller{
		toVolumeId: toVolumeId,
		progress:   newAtomicInt32(0),
		stop:       stop,
		logl:       logex.Levels(logger),
		db:         db,
		diskAccess: diskAccess,
	}

	go c.run(stop)

	return c
}

func (c *Controller) Progress() int {
	return int(c.progress.Get())
}

func (c *Controller) run(stop *stopper.Stopper) {
	defer stop.Done()
	defer c.logl.Info.Println("stopped")

	continueToken := stodb.StartFromFirst

	fiveSeconds := time.NewTicker(5 * time.Second)

	for {
		// give priority to stop signal
		select {
		case <-stop.Signal:
			return
		default:
		}

		select {
		case <-stop.Signal:
			return
		case <-fiveSeconds.C:
			nextContinueToken, err := c.discoverAndRunReplicationJobs(continueToken)
			if err != nil {
				c.logl.Error.Printf("discoverAndRunReplicationJobs: %v", err)
				time.Sleep(3 * time.Second) // to not bombard with errors at full speed
			}

			if bytes.Equal(nextContinueToken, stodb.StartFromFirst) {
				c.progress.Set(100)
			} else if len(nextContinueToken) >= 2 {
				// our database is btree so iteration of blobs is sorted, and keys are hashes
				// with random distribution, so we can estimate progress by looking at first
				// 16 bits of hash
				c.progress.Set(int32(float64(binary.BigEndian.Uint16(nextContinueToken[0:2])) / 65536.0 * 100.0))
			}

			continueToken = nextContinueToken
		}
	}
}

func (c *Controller) discoverAndRunReplicationJobs(continueToken []byte) ([]byte, error) {
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
			c.logl.Debug.Printf(
				"repl %s from %d",
				job.Ref.AsHex(),
				job.FromVolumeId)

			if err := c.replicateJob(job); err != nil {
				c.logl.Error.Printf("replicating blob %s: %v", job.Ref.AsHex(), err)
				time.Sleep(3 * time.Second) // to not bombard with errors at full speed
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
		case <-c.stop.Signal:
			return nextContinueToken, nil
		default:
		}

		jobQueue <- job
	}

	return nextContinueToken, nil
}

func (c *Controller) replicateJob(job *replicationJob) error {
	return c.diskAccess.Replicate(
		context.TODO(),
		job.FromVolumeId,
		c.toVolumeId,
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

	toVolBytes := []byte(fmt.Sprintf("%d", c.toVolumeId))

	nextContinueToken := stodb.StartFromFirst

	return jobs, nextContinueToken, stodb.BlobsPendingReplicationByVolumeIndex.Query(toVolBytes, continueToken, func(id []byte) error {
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

		if !sliceutil.ContainsInt(blob.VolumesPendingReplication, c.toVolumeId) {
			return fmt.Errorf(
				"blob %s volume %d not pending replication (but found from index query)",
				ref.AsHex(),
				c.toVolumeId)
		}

		bestFromVolume, err := c.diskAccess.BestVolumeId(blob.Volumes)
		if err != nil {
			return err
		}

		jobs = append(jobs, &replicationJob{
			Ref:          blob.Ref,
			FromVolumeId: bestFromVolume,
		})

		return nil
	}, tx)
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
