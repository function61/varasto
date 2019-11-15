package stoserver

import (
	"errors"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"strings"
)

type ReplicationPolicyV2 struct {
	Replicas int
}

func (r *ReplicationPolicyV2) Satisfied(currentReplicas int) bool {
	return currentReplicas >= r.Replicas
}

type ReconciliationCompletionReport struct {
	NrCollectionsWithCompliantPolicy  int
	CollectionsWithNonCompliantPolicy []collectionToReconcile
}

type collectionToReconcile struct {
	collectionId    string
	blobCount       int
	desiredReplicas int
	presence        map[int]int
}

var latestReconciliationReport *ReconciliationCompletionReport

func NewReconciliationCompletionReport() *ReconciliationCompletionReport {
	return &ReconciliationCompletionReport{
		CollectionsWithNonCompliantPolicy: []collectionToReconcile{},
	}
}

func (c *cHandlers) VolumeMarkDataLost(cmd *stoservertypes.VolumeMarkDataLost, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		volToPurge, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		volToPurge.BlobSizeTotal = 0
		volToPurge.BlobCount = 0

		if err := stodb.VolumeRepository.Update(volToPurge, tx); err != nil {
			return err
		}

		return stodb.BlobRepository.Each(func(record interface{}) error {
			blob := record.(*stotypes.Blob)

			writtenAndPendingVolumes := func() int {
				return len(blob.Volumes) + len(blob.VolumesPendingReplication)
			}

			volumesBefore := writtenAndPendingVolumes()

			blob.Volumes = sliceutil.FilterInt(blob.Volumes, func(volId int) bool {
				return volId != volToPurge.ID
			})

			blob.VolumesPendingReplication = sliceutil.FilterInt(blob.VolumesPendingReplication, func(volId int) bool {
				return volId != volToPurge.ID
			})

			// optimization to not save unchanged
			if volumesBefore == writtenAndPendingVolumes() { // volume purge did not affect this?
				return nil
			}

			if cmd.OnlyIfRedundancy && len(blob.Volumes) == 0 {
				return errors.New("aborting because blob would lose last redundant copy")
			}

			return stodb.BlobRepository.Update(blob, tx)
		}, tx)
	})
}

func (c *cHandlers) DatabaseReconcileReplicationPolicy(cmd *stoservertypes.DatabaseReconcileReplicationPolicy, ctx *command.Ctx) error {
	collIds := strings.Split(cmd.Id, ",")

	if err := c.db.Update(func(tx *bolt.Tx) error {
		targetVol, err := stodb.Read(tx).Volume(cmd.Volume)
		if err != nil {
			return err
		}

		for _, collId := range collIds {
			coll, err := stodb.Read(tx).Collection(collId)
			if err != nil {
				return err
			}

			if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
				blob, err := stodb.Read(tx).Blob(ref)
				if err != nil {
					return err
				}

				if sliceutil.ContainsInt(blob.Volumes, targetVol.ID) || sliceutil.ContainsInt(blob.VolumesPendingReplication, targetVol.ID) {
					return nil // nothing to do
				}

				blob.VolumesPendingReplication = append(blob.VolumesPendingReplication, targetVol.ID)

				if err := stodb.BlobRepository.Update(blob, tx); err != nil {
					return err
				}

				return nil
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if latestReconciliationReport == nil {
		return nil
	}

	removed := []collectionToReconcile{}
	for _, item := range latestReconciliationReport.CollectionsWithNonCompliantPolicy {
		if !sliceutil.ContainsString(collIds, item.collectionId) {
			removed = append(removed, item)
		}
	}

	latestReconciliationReport.CollectionsWithNonCompliantPolicy = removed

	return nil
}

func (c *cHandlers) DatabaseDiscoverReconcilableReplicationPolicies(cmd *stoservertypes.DatabaseDiscoverReconcilableReplicationPolicies, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rcr := NewReconciliationCompletionReport()

	replicationPolicyForCollection := func(coll *stotypes.Collection) ReplicationPolicyV2 {
		// TODO
		return ReplicationPolicyV2{
			Replicas: 2,
		}
	}

	visitCollection := func(coll *stotypes.Collection) error {
		policy := replicationPolicyForCollection(coll)

		ctr := collectionToReconcile{
			collectionId:    coll.ID,
			desiredReplicas: policy.Replicas,
			presence:        map[int]int{},
			blobCount:       0,
		}

		if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
			ctr.blobCount++

			blob, err := stodb.Read(tx).Blob(ref)
			if err != nil {
				return err
			}

			volsAndPendings := append(blob.Volumes, blob.VolumesPendingReplication...)

			for _, vol := range volsAndPendings {
				// null value (when not found from map) conveniently works out for us
				presence := ctr.presence[vol]
				presence++
				ctr.presence[vol] = presence
			}

			return nil
		}); err != nil {
			return err
		}

		fullReplicas := 0

		for _, blobCount := range ctr.presence {
			if blobCount == ctr.blobCount {
				fullReplicas++
			}
		}

		// empty collections (blobCount=0) are stupid but technically they're fully replicated
		if ctr.blobCount > 0 && !policy.Satisfied(fullReplicas) {
			rcr.CollectionsWithNonCompliantPolicy = append(rcr.CollectionsWithNonCompliantPolicy, ctr)
		} else {
			rcr.NrCollectionsWithCompliantPolicy++
		}

		return nil
	}

	var iterateDirectoryRecursively func(string) error

	iterateDirectoryRecursively = func(id string) error {
		colls, err := stodb.Read(tx).CollectionsByDirectory(id)
		if err != nil {
			return err
		}

		for _, coll := range colls {
			if err := visitCollection(&coll); err != nil {
				return err
			}
		}

		subDirs, err := stodb.Read(tx).SubDirectories(id)
		if err != nil {
			return err
		}

		for _, subDir := range subDirs {
			if err := iterateDirectoryRecursively(subDir.ID); err != nil {
				return err
			}
		}

		return nil
	}

	// start from root and recurse to subdirs
	if err := iterateDirectoryRecursively(stoservertypes.RootFolderId); err != nil {
		return err
	}

	latestReconciliationReport = rcr

	return nil
}

func (c *cHandlers) DatabaseScanAbandoned(cmd *stoservertypes.DatabaseScanAbandoned, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	logl := logex.Levels(logex.Prefix("abandonedscanner", c.logger))

	blobCount := 0
	totalSize := int64(0)

	knownEncryptionKeys := map[string]bool{}

	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		for _, encryptionKey := range coll.EncryptionKeys {
			knownEncryptionKeys[encryptionKey.KeyId] = true
		}

		return nil
	}, tx); err != nil {
		return err
	}

	if err := stodb.BlobRepository.Each(func(record interface{}) error {
		blob := record.(*stotypes.Blob)

		blobCount++
		totalSize += int64(blob.Size)

		if len(blob.Volumes) == 0 {
			logl.Error.Printf("Blob[%s] without a volume", blob.Ref.AsHex())
		}

		if len(blob.Crc32) == 0 {
			logl.Error.Printf("Blob[%s] without Crc32", blob.Ref.AsHex())
		}

		if _, known := knownEncryptionKeys[blob.EncryptionKeyId]; !known {
			logl.Error.Printf(
				"Blob[%s] refers to unknown EncryptionKeyId[%s]",
				blob.Ref.AsHex(),
				blob.EncryptionKeyId)
		}

		return nil
	}, tx); err != nil {
		return err
	}

	logl.Info.Printf("Completed with %d blob(s) with total size (not counting redundancy) %d byte(s) scanned", blobCount, totalSize)

	return nil
}

// FIXME: is this already implemented somewhere
func eachBlobOfCollection(coll *stotypes.Collection, visit func(ref stotypes.BlobRef) error) error {
	visitBlobRefs := func(brs []string) error {
		for _, brSerialized := range brs {
			br, err := stotypes.BlobRefFromHex(brSerialized)
			if err != nil {
				return err
			}

			if err := visit(*br); err != nil {
				return err
			}
		}

		return nil
	}

	for _, changeset := range coll.Changesets {
		for _, created := range changeset.FilesCreated {
			if err := visitBlobRefs(created.BlobRefs); err != nil {
				return err
			}
		}

		for _, updated := range changeset.FilesUpdated {
			if err := visitBlobRefs(updated.BlobRefs); err != nil {
				return err
			}
		}
	}

	return nil
}
