// Controls replication of data between volumes
package storeplication

import (
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
	"time"
)

type replicationJob struct {
	Ref          stotypes.BlobRef
	FromVolumeId int
}

type controller struct {
	toVolumeId int
	stop       *stopper.Stopper
	logl       *logex.Leveled
	db         *bolt.DB
	diskAccess *stodiskaccess.Controller
}

func StartReplicationController(toVolumeId int, db *bolt.DB, diskAccess *stodiskaccess.Controller, logger *log.Logger, stop *stopper.Stopper) {
	logl := logex.Levels(logger)

	defer stop.Done()
	defer logl.Info.Println("stopped")

	fiveSeconds := time.NewTicker(5 * time.Second)

	c := &controller{
		toVolumeId: toVolumeId,
		stop:       stop,
		logl:       logl,
		db:         db,
		diskAccess: diskAccess,
	}

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
			if err := c.discoverAndRunReplicationJobs(); err != nil {
				logl.Error.Printf("discoverAndRunReplicationJobs: %v", err)
				time.Sleep(3 * time.Second) // to not bombard with errors at full speed
			}
		}
	}
}

func (c *controller) discoverAndRunReplicationJobs() error {
	jobs, err := c.discoverReplicationJobs()
	if err != nil {
		return err
	}

	// cap is the amount of runners we'll spawn
	jobQueue := make(chan *replicationJob, 3)

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
			return nil
		default:
		}

		jobQueue <- job
	}

	return nil
}

func (c *controller) replicateJob(job *replicationJob) error {
	return c.diskAccess.Replicate(
		job.FromVolumeId,
		c.toVolumeId,
		job.Ref)
}

func (c *controller) discoverReplicationJobs() ([]*replicationJob, error) {
	tx, err := c.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	batchLimit := 500

	jobs := []*replicationJob{}

	toVolBytes := []byte(fmt.Sprintf("%d", c.toVolumeId))

	return jobs, stodb.BlobsPendingReplicationByVolumeIndex.Query(toVolBytes, stodb.StartFromFirst, func(id []byte) error {
		if len(jobs) == batchLimit {
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
