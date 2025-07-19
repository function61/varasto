// Encapsulates access to the metadata database
package stodb

import (
	"encoding/binary"
	"fmt"

	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stotypes"
)

// re-export so not all stodb-importing packages have to import blorm
var (
	StartFromFirst = blorm.StartFromFirst
	StopIteration  = blorm.ErrStopIteration
)

var BlobRepository = register("Blob", blorm.NewSimpleRepo(
	"blobs",
	func() any { return &stotypes.Blob{} },
	func(record any) []byte { return record.(*stotypes.Blob).Ref }))

var BlobsPendingReplicationByVolumeIndex = blorm.NewValueIndex("repl_pend", BlobRepository, func(record any, index func(val []byte)) {
	blob := record.(*stotypes.Blob)

	for _, volID := range blob.VolumesPendingReplication {
		index([]byte(fmt.Sprintf("%d", volID)))
	}
})

var NodeRepository = register("Node", blorm.NewSimpleRepo(
	"nodes",
	func() any { return &stotypes.Node{} },
	func(record any) []byte { return []byte(record.(*stotypes.Node).ID) }))

var ClientRepository = register("Client", blorm.NewSimpleRepo(
	"clients",
	func() any { return &stotypes.Client{} },
	func(record any) []byte { return []byte(record.(*stotypes.Client).ID) }))

var KeyEncryptionKeyRepository = register("KeyEncryptionKey", blorm.NewSimpleRepo(
	"keyencryptionkeys",
	func() any { return &stotypes.KeyEncryptionKey{} },
	func(record any) []byte { return []byte(record.(*stotypes.KeyEncryptionKey).ID) }))

var ReplicationPolicyRepository = register("ReplicationPolicy", blorm.NewSimpleRepo(
	"replicationpolicies",
	func() any { return &stotypes.ReplicationPolicy{} },
	func(record any) []byte { return []byte(record.(*stotypes.ReplicationPolicy).ID) }))

var VolumeRepository = register("Volume", blorm.NewSimpleRepo(
	"volumes",
	func() any { return &stotypes.Volume{} },
	func(record any) []byte {
		return volumeIntIDToBytes(record.(*stotypes.Volume).ID)
	}))

var VolumeMountRepository = register("VolumeMount", blorm.NewSimpleRepo(
	"volumemounts",
	func() any { return &stotypes.VolumeMount{} },
	func(record any) []byte { return []byte(record.(*stotypes.VolumeMount).ID) }))

var DirectoryRepository = register("Directory", blorm.NewSimpleRepo(
	"directories",
	func() any { return &stotypes.Directory{} },
	func(record any) []byte { return []byte(record.(*stotypes.Directory).ID) }))

var SubdirectoriesIndex = blorm.NewValueIndex("parent", DirectoryRepository, func(record any, index func(val []byte)) {
	dir := record.(*stotypes.Directory)

	if dir.Parent != "" {
		index([]byte(dir.Parent))
	}
})

var CollectionRepository = register("Collection", blorm.NewSimpleRepo(
	"collections",
	func() any { return &stotypes.Collection{} },
	func(record any) []byte { return []byte(record.(*stotypes.Collection).ID) }))

var CollectionsByDataEncryptionKeyIndex = blorm.NewValueIndex("dek", CollectionRepository, func(record any, index func(val []byte)) {
	coll := record.(*stotypes.Collection)

	for _, dekEnvelopes := range coll.EncryptionKeys {
		index([]byte(dekEnvelopes.KeyID))
	}
})

var CollectionsByDirectoryIndex = blorm.NewValueIndex("directory", CollectionRepository, func(record any, index func(val []byte)) {
	coll := record.(*stotypes.Collection)

	index([]byte(coll.Directory))
})

var CollectionsGlobalVersionIndex = blorm.NewRangeIndex("globalversion", CollectionRepository, func(record any, index func(sortKey []byte)) {
	coll := record.(*stotypes.Collection)

	globalVersion := make([]byte, 8)
	binary.BigEndian.PutUint64(globalVersion, coll.GlobalVersion)

	index(globalVersion)
})

var IntegrityVerificationJobRepository = register("IntegrityVerificationJob", blorm.NewSimpleRepo(
	"ivjobs",
	func() any { return &stotypes.IntegrityVerificationJob{} },
	func(record any) []byte { return []byte(record.(*stotypes.IntegrityVerificationJob).ID) }))

var ScheduledJobRepository = register("ScheduledJob", blorm.NewSimpleRepo(
	"scheduledjobs",
	func() any { return &stotypes.ScheduledJob{} },
	func(record any) []byte { return []byte(record.(*stotypes.ScheduledJob).ID) }))

var configRepository = register("Config", blorm.NewSimpleRepo(
	"config",
	func() any { return &stotypes.Config{} },
	func(record any) []byte { return []byte(record.(*stotypes.Config).Key) }))

// helpers

func volumeIntIDToBytes(id int) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(id))
	return b
}

// appenders. Go surely would need some generic love..

func ClientAppender(slice *[]stotypes.Client) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.Client))
		return nil
	}
}

func NodeAppender(slice *[]stotypes.Node) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.Node))
		return nil
	}
}

func ReplicationPolicyAppender(slice *[]stotypes.ReplicationPolicy) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.ReplicationPolicy))
		return nil
	}
}

func VolumeAppender(slice *[]stotypes.Volume) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.Volume))
		return nil
	}
}

func VolumeMountAppender(slice *[]stotypes.VolumeMount) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.VolumeMount))
		return nil
	}
}

func IntegrityVerificationJobAppender(slice *[]stotypes.IntegrityVerificationJob) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.IntegrityVerificationJob))
		return nil
	}
}

func ScheduledJobAppender(slice *[]stotypes.ScheduledJob) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.ScheduledJob))
		return nil
	}
}

func KeyEncryptionKeyAppender(slice *[]stotypes.KeyEncryptionKey) func(record any) error {
	return func(record any) error {
		*slice = append(*slice, *record.(*stotypes.KeyEncryptionKey))
		return nil
	}
}

// key is heading in export file under which all JSON records are dumped
var RepoByRecordType = map[string]blorm.Repository{}

// register known repo for exporting
func register(exportImportKey string, repo *blorm.SimpleRepository) *blorm.SimpleRepository {
	RepoByRecordType[exportImportKey] = repo
	return repo
}
