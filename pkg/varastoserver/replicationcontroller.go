package varastoserver

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/varastotypes"
	"go.etcd.io/bbolt"
	"log"
	"sync"
	"time"
)

type replicationJob struct {
	Ref          varastotypes.BlobRef
	FromVolumeId int
	ToVolumeId   int
}

func StartReplicationController(db *bolt.DB, serverConfig *ServerConfig, logger *log.Logger, stop *stopper.Stopper) {
	logl := logex.Levels(logger)

	defer stop.Done()
	defer logl.Info.Println("stopped")

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
			if err := discoverAndRunReplicationJobs(db, logl, serverConfig, stop); err != nil {
				logl.Error.Printf("discoverAndRunReplicationJobs: %v", err)
				time.Sleep(3 * time.Second) // to not bombard with errors at full speed
			}
		}
	}
}

func discoverAndRunReplicationJobs(
	db *bolt.DB,
	logl *logex.Leveled,
	serverConfig *ServerConfig,
	stop *stopper.Stopper,
) error {
	jobs, err := discoverReplicationJobs(db, logl, serverConfig)
	if err != nil {
		return err
	}

	// cap is the amount of runners we'll spawn
	jobQueue := make(chan *replicationJob, 3)

	jobRunnersDone := sync.WaitGroup{}

	runner := func() {
		defer jobRunnersDone.Done()

		for job := range jobQueue {
			logl.Debug.Printf(
				"repl %s from %d -> %d",
				job.Ref.AsHex(),
				job.FromVolumeId,
				job.ToVolumeId)

			if err := replicateJob(job, db, serverConfig); err != nil {
				logl.Error.Printf("replicating blob %s: %v", job.Ref.AsHex(), err)
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
		case <-stop.Signal:
			return nil
		default:
		}

		jobQueue <- job
	}

	return nil
}

func replicateJob(job *replicationJob, db *bolt.DB, serverConfig *ServerConfig) error {
	return serverConfig.DiskAccess.Replicate(
		job.FromVolumeId,
		job.ToVolumeId,
		job.Ref)
}

func discoverReplicationJobs(db *bolt.DB, logl *logex.Leveled, serverConfig *ServerConfig) ([]*replicationJob, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blobsPendingReplication := BlobsPendingReplicationIndex.Bucket(tx)

	batchLimit := 100

	jobs := []*replicationJob{}

	all := blobsPendingReplication.Cursor()
	for key, _ := all.First(); key != nil; key, _ = all.Next() {
		if len(jobs) == batchLimit {
			logl.Info.Printf(
				"operating @ batchLimit (%d)",
				batchLimit)
			break
		}

		ref := varastotypes.BlobRef(key)

		blob, err := QueryWithTx(tx).Blob(ref)
		if err != nil {
			return nil, err
		}

		for _, toVolumeId := range blob.VolumesPendingReplication {
			bestVolume, err := serverConfig.DiskAccess.BestVolumeId(blob.Volumes)
			if err != nil {
				return nil, err
			}

			jobs = append(jobs, &replicationJob{
				Ref:          blob.Ref,
				FromVolumeId: bestVolume,
				ToVolumeId:   toVolumeId,
			})
		}
	}

	return jobs, nil
}
