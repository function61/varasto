package stoserver

import (
	"errors"
	"fmt"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/byteshuman"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/storeplication"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

type ReconciliationCompletionReport struct {
	Timestamp                         time.Time // of report start (because that's what the tx sees)
	TotalCollections                  int
	EmptyCollectionIds                []string
	EmptyDirectoryIds                 []string
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
		EmptyCollectionIds:                []string{},
		EmptyDirectoryIds:                 []string{},
		CollectionsWithNonCompliantPolicy: []collectionToReconcile{},
	}
}

func (c *cHandlers) VolumeDecommission(cmd *stoservertypes.VolumeDecommission, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		if vol.BlobCount != 0 {
			return fmt.Errorf(
				"Refusing to decommission non-empty volume (has %d blobs) for your safety. Mark data lost first!",
				vol.BlobCount)
		}

		if vol.SmartId != "" {
			return errors.New("volume still has SMART polling enabled")
		}

		if err := stodb.VolumeMountRepository.Each(func(record interface{}) error {
			mount := record.(*stotypes.VolumeMount)

			if mount.Volume == vol.ID {
				return fmt.Errorf("volume is still mounted (mount ID %s)", mount.ID)
			}

			return nil
		}, tx); err != nil {
			return err
		}

		if err := stodb.ReplicationPolicyRepository.Each(func(record interface{}) error {
			policy := record.(*stotypes.ReplicationPolicy)

			if sliceutil.ContainsInt(policy.DesiredVolumes, vol.ID) {
				return fmt.Errorf(
					"Policy '%s' still wants to write data to the volume you're trying to decommission",
					policy.Name)
			}

			return nil
		}, tx); err != nil {
			return err
		}

		hasQueuedWrites, err := storeplication.HasQueuedWriteIOsForVolume(vol.ID, tx)
		if err != nil {
			return err
		}

		if hasQueuedWrites {
			return fmt.Errorf("volume %s has queued write I/Os", vol.Label)
		}

		vol.Decommissioned = &ctx.Meta.Timestamp
		vol.DecommissionReason = cmd.Reason

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeRemoveQueuedReplications(cmd *stoservertypes.VolumeRemoveQueuedReplications, ctx *command.Ctx) error {
	from := cmd.From // shorthand

	if _, hasReplicationController := c.conf.ReplicationControllers[from]; hasReplicationController {
		// no other danger but the controller reads a batch of work (blob refs) into memory,
		// so it could operate on canceled work for a while if we let this happen
		return fmt.Errorf(
			"Volume %d has replication controller running. Please stop it first, e.g. by unmounting the volume.",
			from)
	}

	totalBlobs := 0
	removed := 0

	if err := c.db.Update(func(tx *bbolt.Tx) error {
		return stodb.BlobRepository.Each(func(record interface{}) error {
			blob := record.(*stotypes.Blob)

			totalBlobs++

			if !sliceutil.ContainsInt(blob.VolumesPendingReplication, from) {
				return nil
			}

			removed++

			blob.VolumesPendingReplication = sliceutil.FilterInt(blob.VolumesPendingReplication, func(vol int) bool {
				return vol != from
			})

			return stodb.BlobRepository.Update(blob, tx)
		}, tx)
	}); err != nil {
		return err
	}

	if removed == 0 {
		return fmt.Errorf(
			"Volume %d (with %d blobs) didn't have any queued replications",
			from,
			totalBlobs)
	}

	logex.Levels(c.logger).Info.Printf(
		"VolumeRemoveQueuedReplications %d/%d",
		removed,
		totalBlobs)

	return nil
}

func (c *cHandlers) VolumeMarkDataLost(cmd *stoservertypes.VolumeMarkDataLost, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		volToPurge, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		// make sure no new data will be landing on this volume
		if err := noReplicationPolicyPlacesNewDataToVolume(volToPurge.ID, tx); err != nil {
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
	collIds := *cmd.Collections

	if err := c.db.Update(func(tx *bbolt.Tx) error {
		volumeById, err := buildVolumeByIdLookup(tx)
		if err != nil {
			return err
		}

		targetVol, err := stodb.Read(tx).Volume(cmd.Volume)
		if err != nil {
			return err
		}

		for _, collId := range collIds {
			coll, err := stodb.Read(tx).Collection(collId)
			if err != nil {
				return err
			}

			policy, err := stodb.Read(tx).ReplicationPolicy(coll.ReplicationPolicy)
			if err != nil {
				return err
			}

			if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
				blob, err := stodb.Read(tx).Blob(ref)
				if err != nil {
					return err
				}

				problemRedundancy, problemZoning := blobProblems(blob, policy, volumeById)

				if !problemRedundancy && !problemZoning {
					return nil // nothing to fix
				}

				volsAndPendings := append([]int{}, blob.Volumes...)
				volsAndPendings = append(volsAndPendings, blob.VolumesPendingReplication...)

				if sliceutil.ContainsInt(volsAndPendings, targetVol.ID) {
					return nil // blob already exists (or does soon) in this volume
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

		if cmd.Policy != "" { // validation
			if _, err := stodb.Read(tx).ReplicationPolicy(cmd.Policy); err != nil {
				return err
			}
		} else { // unsetting policy
			if dir.Parent == "" {
				return errors.New("cannot unset replication policy from root directory")
			}
		}

		dir.ReplicationPolicy = cmd.Policy

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

	notBrandNew := func(ts time.Time) bool {
		return time.Since(ts) > 2*time.Hour
	}

	logl := logex.Levels(logex.Prefix("reconciliation", c.logger))

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
	volumeById, err := buildVolumeByIdLookup(tx)
	if err != nil {
		return err
	}

	fixedReplPolicies := 0

	report := NewReconciliationCompletionReport()

	visitCollection := func(coll *stotypes.Collection, effectiveReplPolicyId string) error {
		report.TotalCollections++

		policy := policyById[effectiveReplPolicyId]
		if policy == nil { // should not happen
			return fmt.Errorf("policy not found: %s", effectiveReplPolicyId)
		}

		// don't count "dir meta" collections, as they are allowed to be empty
		if len(coll.Changesets) == 0 && notBrandNew(coll.Created) && coll.Name != stoservertypes.StoDirMetaName {
			report.EmptyCollectionIds = append(report.EmptyCollectionIds, coll.ID)
		}

		if coll.ReplicationPolicy != effectiveReplPolicyId {
			coll.ReplicationPolicy = effectiveReplPolicyId

			if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
				return err
			}

			fixedReplPolicies++
		}

		collReport, err := reconciliationReportForCollection(coll, policy, volumeById, tx)
		if err != nil {
			return err
		}

		if collReport.anyProblems() {
			report.CollectionsWithNonCompliantPolicy = append(report.CollectionsWithNonCompliantPolicy, *collReport)
		}

		return nil
	}

	// start from root and recurse to subdirs
	root, err := stodb.Read(tx).Directory(stoservertypes.RootFolderId)
	if err != nil {
		return err
	}

	if err := iterateDirectoriesRecursively(root, "", tx, func(dir *stotypes.Directory, colls []stotypes.Collection, numSubdirs int, effectiveReplPolicyId string) error {
		if len(colls)+numSubdirs == 0 && notBrandNew(dir.Created) {
			report.EmptyDirectoryIds = append(report.EmptyDirectoryIds, dir.ID)
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
		logl.Info.Printf("fixed %d replication policies", fixedReplPolicies)
	}

	latestReconciliationReport = report

	return tx.Commit()
}

// detect problems for each blob of collection and wrap it in a reconciliation report for
// UI (how many desired replicas, which volumes have blobs for this collection etc.)
func reconciliationReportForCollection(
	coll *stotypes.Collection,
	policy *stotypes.ReplicationPolicy,
	volumeById map[int]*stotypes.Volume,
	tx *bbolt.Tx,
) (*collectionToReconcile, error) {
	collReport := &collectionToReconcile{
		collectionId:    coll.ID,
		desiredReplicas: len(policy.DesiredVolumes),
		presence:        map[int]int{},
		blobCount:       0,
	}

	if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
		collReport.blobCount++

		blob, err := stodb.Read(tx).Blob(ref)
		if err != nil {
			return err
		}

		problemRedundancy, problemZoning := blobProblems(blob, policy, volumeById)

		for _, vol := range append(blob.Volumes, blob.VolumesPendingReplication...) {
			// zero value (when not found from map) conveniently works out for us
			collReport.presence[vol]++
		}

		if problemRedundancy {
			collReport.problemRedundancy = true
		}

		if problemZoning {
			collReport.problemZoning = true
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return collReport, nil
}

func (c *cHandlers) DatabaseScanAbandoned(cmd *stoservertypes.DatabaseScanAbandoned, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	logl := logex.Levels(logex.Prefix("abandonedscanner", c.logger))

	blobCount := 0
	totalSize := uint64(0)

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
		totalSize += uint64(blob.Size)

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

	logl.Info.Printf(
		"Completed with %d blob(s) with total size (not counting redundancy) %d scanned",
		blobCount,
		byteshuman.Humanize(totalSize))

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
	visitor func(dir *stotypes.Directory, colls []stotypes.Collection, numSubdirs int, effectiveReplPolicyId string) error,
) error {
	if dir.ReplicationPolicy != "" {
		effectiveReplPolicyId = dir.ReplicationPolicy
	}

	colls, err := stodb.Read(tx).CollectionsByDirectory(dir.ID)
	if err != nil {
		return err
	}

	subDirs, err := stodb.Read(tx).SubDirectories(dir.ID)
	if err != nil {
		return err
	}

	if err := visitor(dir, colls, len(subDirs), effectiveReplPolicyId); err != nil {
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

func noReplicationPolicyPlacesNewDataToVolume(volId int, tx *bbolt.Tx) error {
	return stodb.ReplicationPolicyRepository.Each(func(record interface{}) error {
		policy := record.(*stotypes.ReplicationPolicy)

		if sliceutil.ContainsInt(policy.DesiredVolumes, volId) {
			return fmt.Errorf(
				"Replication policy '%s' sends new data to your volume",
				policy.Name)
		}

		return nil
	}, tx)
}

// check if blob has redundancy or zoning problems according to a given policy
func blobProblems(
	blob *stotypes.Blob,
	policy *stotypes.ReplicationPolicy,
	volumeById map[int]*stotypes.Volume,
) (bool, bool) {
	volsAndPendings := append([]int{}, blob.Volumes...)
	volsAndPendings = append(volsAndPendings, blob.VolumesPendingReplication...)
	uniqueZones := map[string]bool{}
	for _, vol := range volsAndPendings {
		uniqueZones[volumeById[vol].Zone] = true
	}

	problemRedundancy := len(volsAndPendings) < policy.ReplicaCount()
	problemZoning := len(uniqueZones) < policy.MinZones

	return problemRedundancy, problemZoning
}

func buildVolumeByIdLookup(tx *bbolt.Tx) (map[int]*stotypes.Volume, error) {
	volumeById := map[int]*stotypes.Volume{}

	if err := stodb.VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)
		volumeById[vol.ID] = vol
		return nil
	}, tx); err != nil {
		return nil, err
	}

	return volumeById, nil
}
