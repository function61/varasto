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
	StopIteration  = blorm.StopIteration
)

var BlobRepository = register("Blob", blorm.NewSimpleRepo(
	"blobs",
	func() interface{} { return &stotypes.Blob{} },
	func(record interface{}) []byte { return record.(*stotypes.Blob).Ref }))

var BlobsPendingReplicationByVolumeIndex = blorm.NewValueIndex("repl_pend", BlobRepository, func(record interface{}, index func(val []byte)) {
	blob := record.(*stotypes.Blob)

	for _, volId := range blob.VolumesPendingReplication {
		index([]byte(fmt.Sprintf("%d", volId)))
	}
})

var NodeRepository = register("Node", blorm.NewSimpleRepo(
	"nodes",
	func() interface{} { return &stotypes.Node{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Node).ID) }))

var ClientRepository = register("Client", blorm.NewSimpleRepo(
	"clients",
	func() interface{} { return &stotypes.Client{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Client).ID) }))

var KeyEncryptionKeyRepository = register("KeyEncryptionKey", blorm.NewSimpleRepo(
	"keyencryptionkeys",
	func() interface{} { return &stotypes.KeyEncryptionKey{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.KeyEncryptionKey).ID) }))

var ReplicationPolicyRepository = register("ReplicationPolicy", blorm.NewSimpleRepo(
	"replicationpolicies",
	func() interface{} { return &stotypes.ReplicationPolicy{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.ReplicationPolicy).ID) }))

var VolumeRepository = register("Volume", blorm.NewSimpleRepo(
	"volumes",
	func() interface{} { return &stotypes.Volume{} },
	func(record interface{}) []byte {
		return volumeIntIdToBytes(record.(*stotypes.Volume).ID)
	}))

var VolumeMountRepository = register("VolumeMount", blorm.NewSimpleRepo(
	"volumemounts",
	func() interface{} { return &stotypes.VolumeMount{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.VolumeMount).ID) }))

var DirectoryRepository = register("Directory", blorm.NewSimpleRepo(
	"directories",
	func() interface{} { return &stotypes.Directory{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Directory).ID) }))

var SubdirectoriesIndex = blorm.NewValueIndex("parent", DirectoryRepository, func(record interface{}, index func(val []byte)) {
	dir := record.(*stotypes.Directory)

	if dir.Parent != "" {
		index([]byte(dir.Parent))
	}
})

var CollectionRepository = register("Collection", blorm.NewSimpleRepo(
	"collections",
	func() interface{} { return &stotypes.Collection{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Collection).ID) }))

var CollectionsByDirectoryIndex = blorm.NewValueIndex("directory", CollectionRepository, func(record interface{}, index func(val []byte)) {
	coll := record.(*stotypes.Collection)

	index([]byte(coll.Directory))
})

var IntegrityVerificationJobRepository = register("IntegrityVerificationJob", blorm.NewSimpleRepo(
	"ivjobs",
	func() interface{} { return &stotypes.IntegrityVerificationJob{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.IntegrityVerificationJob).ID) }))

var ScheduledJobRepository = register("ScheduledJob", blorm.NewSimpleRepo(
	"scheduledjobs",
	func() interface{} { return &stotypes.ScheduledJob{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.ScheduledJob).ID) }))

var configRepository = register("Config", blorm.NewSimpleRepo(
	"config",
	func() interface{} { return &stotypes.Config{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Config).Key) }))

// helpers

func volumeIntIdToBytes(id int) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(id))
	return b
}

// appenders. Go surely would need some generic love..

func ClientAppender(slice *[]stotypes.Client) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.Client))
		return nil
	}
}

func NodeAppender(slice *[]stotypes.Node) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.Node))
		return nil
	}
}

func ReplicationPolicyAppender(slice *[]stotypes.ReplicationPolicy) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.ReplicationPolicy))
		return nil
	}
}

func VolumeAppender(slice *[]stotypes.Volume) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.Volume))
		return nil
	}
}

func VolumeMountAppender(slice *[]stotypes.VolumeMount) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.VolumeMount))
		return nil
	}
}

func IntegrityVerificationJobAppender(slice *[]stotypes.IntegrityVerificationJob) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.IntegrityVerificationJob))
		return nil
	}
}

func ScheduledJobAppender(slice *[]stotypes.ScheduledJob) func(record interface{}) error {
	return func(record interface{}) error {
		*slice = append(*slice, *record.(*stotypes.ScheduledJob))
		return nil
	}
}

func KeyEncryptionKeyAppender(slice *[]stotypes.KeyEncryptionKey) func(record interface{}) error {
	return func(record interface{}) error {
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
