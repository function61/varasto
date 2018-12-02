package bupserver

import (
	"errors"
	"fmt"
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/gokit/stopper"
	"log"
	"time"
)

type replicationJob struct {
	Ref          buptypes.BlobRef
	FromVolumeId string
	ToVolumeId   string
}

func StartReplicationController(db *storm.DB, volumeDrivers VolumeDriverMap, stop *stopper.Stopper) {
	defer stop.Done()
	defer log.Println("ReplicationController stopped")

	fiveSeconds := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-stop.Signal:
			return
		case <-fiveSeconds.C:
			if err := discoverAndRunReplicationJobs(db, volumeDrivers); err != nil {
				log.Printf("discoverAndRunReplicationJobs: %s", err.Error())
			}
		}
	}
}

func discoverAndRunReplicationJobs(db *storm.DB, volumeDrivers VolumeDriverMap) error {
	jobs, err := discoverReplicationJobs(db)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		log.Printf(
			"replicating %s from %s to %s",
			job.Ref.AsHex(),
			job.FromVolumeId,
			job.ToVolumeId)

		if err := replicateJob(job, db, volumeDrivers); err != nil {
			log.Printf("error replicating blob %s", job.Ref.AsHex())
		}
	}

	return nil
}

func replicateJob(job *replicationJob, db *storm.DB, volumeDrivers VolumeDriverMap) error {
	from, ok := volumeDrivers[job.FromVolumeId]
	if !ok {
		return errors.New("from volume not found from volume drivers")
	}
	to, ok := volumeDrivers[job.ToVolumeId]
	if !ok {
		return errors.New("to volume not found from volume drivers")
	}

	stream, err := from.Fetch(job.Ref)
	if err != nil {
		return err
	}

	if _, err := to.Store(job.Ref, buputils.BlobHashVerifier(stream, job.Ref)); err != nil {
		return err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blobRecord := &buptypes.Blob{}
	if err := tx.One("Ref", job.Ref, blobRecord); err != nil {
		return err
	}

	if contains(blobRecord.Volumes, job.ToVolumeId) {
		return fmt.Errorf(
			"race condition: someone already replicated %s to %s",
			job.Ref.AsHex(),
			job.ToVolumeId)
	}

	blobRecord.Volumes = append(blobRecord.Volumes, job.ToVolumeId)

	// remove succesfully replicated blob from pending list
	blobRecord.VolumesPendingReplication = filter(blobRecord.VolumesPendingReplication, func(volId string) bool {
		return volId != job.ToVolumeId
	})
	blobRecord.IsPendingReplication = len(blobRecord.VolumesPendingReplication) > 0

	// TODO: update volume's blob count and total bytes

	if err := tx.Save(blobRecord); err != nil {
		return err
	}

	return tx.Commit()
}

func discoverReplicationJobs(db *storm.DB) ([]*replicationJob, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	batchLimit := 100
	var blobsNeedingReplication []*buptypes.Blob
	if err := tx.Find("IsPendingReplication", true, &blobsNeedingReplication, storm.Limit(batchLimit)); err != nil {
		if err == storm.ErrNotFound {
			return nil, nil // not an error at all
		}

		return nil, err
	}

	if len(blobsNeedingReplication) == batchLimit {
		log.Printf("discoverReplicationJobs: replication operating @ batchLimit")
	}

	needsReplication := []*replicationJob{}

	for _, blob := range blobsNeedingReplication {
		for _, toVolumeId := range blob.VolumesPendingReplication {
			if len(blob.Volumes) == 0 { // should not happen
				panic("blob does not exist at any volume")
			}

			// FIXME: find first thisnode-mounted volume
			firstVolume := blob.Volumes[0]

			needsReplication = append(needsReplication, &replicationJob{
				Ref:          blob.Ref,
				FromVolumeId: firstVolume,
				ToVolumeId:   toVolumeId,
			})
		}
	}

	return needsReplication, nil
}
