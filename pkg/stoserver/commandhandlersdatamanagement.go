package stoserver

import (
	"errors"
	"strings"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

type ReconciliationCompletionReport struct {
	Timestamp                         time.Time // of report start (because that's what the tx sees)
	TotalCollections                  int
	CollectionsWithNonCompliantPolicy []collectionToReconcile
}

type collectionToReconcile struct {
	collectionId      string
	blobCount         int
	desiredReplicas   int
	problemRedundancy bool
	problemZoning     bool
	presence          map[int]int
}

func (c *collectionToReconcile) anyProblems() bool {
	return c.problemRedundancy || c.problemZoning
}

var latestReconciliationReport *ReconciliationCompletionReport

func NewReconciliationCompletionReport() *ReconciliationCompletionReport {
	return &ReconciliationCompletionReport{
		Timestamp:                         time.Now(),
		CollectionsWithNonCompliantPolicy: []collectionToReconcile{},
	}
}

func (c *cHandlers) VolumeMarkDataLost(cmd *stoservertypes.VolumeMarkDataLost, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
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

	if err := c.db.Update(func(tx *bbolt.Tx) error {
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

func (c *cHandlers) ReplicationpolicyCreate(cmd *stoservertypes.ReplicationpolicyCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		if cmd.MinZones < 1 {
			return errors.New("MinZones cannot be < 1")
		}

		return stodb.ReplicationPolicyRepository.Update(&stotypes.ReplicationPolicy{
			ID:             stoutils.NewReplicationPolicyId(),
			Name:           cmd.Name,
			DesiredVolumes: []int{},
			MinZones:       cmd.MinZones,
		}, tx)
	})
}

func (c *cHandlers) ReplicationpolicyRename(cmd *stoservertypes.ReplicationpolicyRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		policy, err := stodb.Read(tx).ReplicationPolicy(cmd.Id)
		if err != nil {
			return err
		}

		policy.Name = cmd.Name

		return stodb.ReplicationPolicyRepository.Update(policy, tx)
	})
}

func (c *cHandlers) ReplicationpolicyChangeMinZones(cmd *stoservertypes.ReplicationpolicyChangeMinZones, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		policy, err := stodb.Read(tx).ReplicationPolicy(cmd.Id)
		if err != nil {
			return err
		}

		if cmd.MinZones < 1 {
			return errors.New("MinZones must be >= 1")
		}

		policy.MinZones = cmd.MinZones

		return stodb.ReplicationPolicyRepository.Update(policy, tx)
	})
}

func (c *cHandlers) DirectoryChangeReplicationPolicy(cmd *stoservertypes.DirectoryChangeReplicationPolicy, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		if cmd.ReplicationPolicy != "" { // validation
			if _, err := stodb.Read(tx).ReplicationPolicy(cmd.ReplicationPolicy); err != nil {
				return err
			}
		} else { // unsetting policy
			if dir.Parent == "" {
				return errors.New("cannot unset replication policy from root directory")
			}
		}

		dir.ReplicationPolicy = cmd.ReplicationPolicy

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DatabaseDiscoverReconcilableReplicationPolicies(cmd *stoservertypes.DatabaseDiscoverReconcilableReplicationPolicies, ctx *command.Ctx) error {
	// why write tx? we'll update out-of-date effective replication policies
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	// load all policies into memory for quick access
	policyById := map[string]*stotypes.ReplicationPolicy{}

	if err := stodb.ReplicationPolicyRepository.Each(func(record interface{}) error {
		policy := record.(*stotypes.ReplicationPolicy)
		policyById[policy.ID] = policy
		return nil
	}, tx); err != nil {
		return err
	}

	// load all volumes into memory for quick access
	volumeById := map[int]*stotypes.Volume{}

	if err := stodb.VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)
		volumeById[vol.ID] = vol
		return nil
	}, tx); err != nil {
		return err
	}

	fixedReplPolicies := 0

	report := NewReconciliationCompletionReport()

	visitCollection := func(coll *stotypes.Collection, effectiveReplPolicyId string) error {
		report.TotalCollections++

		policy := policyById[effectiveReplPolicyId]
		if policy == nil {
			panic("policy not found") // should not happen
		}

		collReport := collectionToReconcile{
			collectionId:    coll.ID,
			desiredReplicas: len(policy.DesiredVolumes),
			presence:        map[int]int{},
			blobCount:       0,
		}

		if coll.ReplicationPolicy != effectiveReplPolicyId {
			coll.ReplicationPolicy = effectiveReplPolicyId

			if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
				return err
			}

			fixedReplPolicies++
		}

		if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
			collReport.blobCount++

			blob, err := stodb.Read(tx).Blob(ref)
			if err != nil {
				return err
			}

			volsAndPendings := append(blob.Volumes, blob.VolumesPendingReplication...)

			uniqueZones := map[string]bool{}

			for _, vol := range volsAndPendings {
				// zero value (when not found from map) conveniently works out for us
				collReport.presence[vol] = collReport.presence[vol] + 1

				uniqueZones[volumeById[vol].Zone] = true
			}

			if len(volsAndPendings) < policy.ReplicaCount() {
				collReport.problemRedundancy = true
			}

			if len(uniqueZones) < policy.MinZones {
				collReport.problemZoning = true
			}

			return nil
		}); err != nil {
			return err
		}

		if collReport.anyProblems() {
			report.CollectionsWithNonCompliantPolicy = append(report.CollectionsWithNonCompliantPolicy, collReport)
		}

		return nil
	}

	// start from root and recurse to subdirs
	root, err := stodb.Read(tx).Directory(stoservertypes.RootFolderId)
	if err != nil {
		return err
	}

	if err := iterateDirectoriesRecursively(root, "", tx, func(dir *stotypes.Directory, effectiveReplPolicyId string) error {
		colls, err := stodb.Read(tx).CollectionsByDirectory(dir.ID)
		if err != nil {
			return err
		}

		for _, coll := range colls {
			coll := coll // pin
			if err := visitCollection(&coll, effectiveReplPolicyId); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if fixedReplPolicies > 0 {
		logex.Levels(logex.Prefix("reconciliation", c.logger)).Info.Printf("fixed %d replication policies", fixedReplPolicies)
	}

	latestReconciliationReport = report

	return tx.Commit()
}

func (c *cHandlers) DatabaseScanAbandoned(cmd *stoservertypes.DatabaseScanAbandoned, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

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

func iterateDirectoriesRecursively(
	dir *stotypes.Directory,
	effectiveReplPolicyId string,
	tx *bbolt.Tx,
	visitor func(dir *stotypes.Directory, effectiveReplPolicyId string) error,
) error {
	if dir.ReplicationPolicy != "" {
		effectiveReplPolicyId = dir.ReplicationPolicy
	}

	if err := visitor(dir, effectiveReplPolicyId); err != nil {
		return err
	}

	subDirs, err := stodb.Read(tx).SubDirectories(dir.ID)
	if err != nil {
		return err
	}

	for _, subDir := range subDirs {
		subDir := subDir // pin

		if err := iterateDirectoriesRecursively(
			&subDir,
			effectiveReplPolicyId,
			tx,
			visitor,
		); err != nil {
			return err
		}
	}

	return nil
}
