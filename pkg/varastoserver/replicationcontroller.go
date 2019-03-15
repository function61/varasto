package varastoserver

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/sliceutil"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"go.etcd.io/bbolt"
	"log"
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
			if err := discoverAndRunReplicationJobs(db, logl, serverConfig); err != nil {
				logl.Error.Printf("discoverAndRunReplicationJobs: %v", err)
			}
		}
	}
}

func discoverAndRunReplicationJobs(db *bolt.DB, logl *logex.Leveled, serverConfig *ServerConfig) error {
	jobs, err := discoverReplicationJobs(db, logl)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		logl.Debug.Printf(
			"replicating %s from %d to %d",
			job.Ref.AsHex(),
			job.FromVolumeId,
			job.ToVolumeId)

		if err := replicateJob(job, db, serverConfig); err != nil {
			logl.Error.Printf("replicating blob %s", job.Ref.AsHex())
		}
	}

	return nil
}

func replicateJob(job *replicationJob, db *bolt.DB, serverConfig *ServerConfig) error {
	from, ok := serverConfig.VolumeDrivers[job.FromVolumeId]
	if !ok {
		return errors.New("from volume not found from volume drivers")
	}
	to, ok := serverConfig.VolumeDrivers[job.ToVolumeId]
	if !ok {
		return errors.New("to volume not found from volume drivers")
	}

	stream, err := from.Fetch(job.Ref)
	if err != nil {
		return err
	}

	blobSizeBytes, err := to.Store(job.Ref, varastoutils.BlobHashVerifier(stream, job.Ref))
	if err != nil {
		return err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blobRecord, err := QueryWithTx(tx).Blob(job.Ref)
	if err != nil {
		return err
	}

	if sliceutil.ContainsInt(blobRecord.Volumes, job.ToVolumeId) {
		return fmt.Errorf(
			"race condition: someone already replicated %s to %d",
			job.Ref.AsHex(),
			job.ToVolumeId)
	}

	blobRecord.Volumes = append(blobRecord.Volumes, job.ToVolumeId)

	// remove succesfully replicated volumes from pending list
	blobRecord.VolumesPendingReplication = sliceutil.FilterInt(blobRecord.VolumesPendingReplication, func(volId int) bool {
		return volId != job.ToVolumeId
	})
	blobRecord.IsPendingReplication = len(blobRecord.VolumesPendingReplication) > 0

	if err := volumeManagerIncreaseBlobCount(tx, job.ToVolumeId, blobSizeBytes); err != nil {
		return err
	}

	if err := BlobRepository.Update(blobRecord, tx); err != nil {
		return err
	}

	return tx.Commit()
}

func discoverReplicationJobs(db *bolt.DB, logl *logex.Leveled) ([]*replicationJob, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	return []*replicationJob{}, nil
	/*
		batchLimit := 100
		var blobsNeedingReplication []*varastotypes.Blob
		if err := tx.Find("IsPendingReplication", true, &blobsNeedingReplication, storm.Limit(batchLimit)); err != nil {
			if err == storm.ErrNotFound {
				return nil, nil // not an error at all
			}

			return nil, err
		}

		if len(blobsNeedingReplication) == batchLimit {
			logl.Info.Printf(
				"operating @ batchLimit (%d)",
				batchLimit)
		}

		jobs := []*replicationJob{}

		for _, blob := range blobsNeedingReplication {
			for _, toVolumeId := range blob.VolumesPendingReplication {
				if len(blob.Volumes) == 0 { // should not happen
					panic("blob does not exist at any volume")
				}

				// FIXME: find first thisnode-mounted volume
				firstVolume := blob.Volumes[0]

				jobs = append(jobs, &replicationJob{
					Ref:          blob.Ref,
					FromVolumeId: firstVolume,
					ToVolumeId:   toVolumeId,
				})
			}
		}

		return jobs, nil
	*/
}
