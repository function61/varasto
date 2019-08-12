// Encapsulates access to the metadata database
package stodb

import (
	"encoding/binary"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stotypes"
)

// re-export so not all stodb-importing packages have to import blorm
var (
	StartFromFirst = blorm.StartFromFirst
	StopIteration  = blorm.StopIteration
)

var BlobRepository = blorm.NewSimpleRepo(
	"blobs",
	func() interface{} { return &stotypes.Blob{} },
	func(record interface{}) []byte { return record.(*stotypes.Blob).Ref })

var BlobsPendingReplicationIndex = blorm.NewSetIndex("pending_replication", BlobRepository, func(record interface{}) bool {
	blob := record.(*stotypes.Blob)

	return len(blob.VolumesPendingReplication) > 0
})

var NodeRepository = blorm.NewSimpleRepo(
	"nodes",
	func() interface{} { return &stotypes.Node{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Node).ID) })

var ClientRepository = blorm.NewSimpleRepo(
	"clients",
	func() interface{} { return &stotypes.Client{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Client).ID) })

var ReplicationPolicyRepository = blorm.NewSimpleRepo(
	"replicationpolicies",
	func() interface{} { return &stotypes.ReplicationPolicy{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.ReplicationPolicy).ID) })

var VolumeRepository = blorm.NewSimpleRepo(
	"volumes",
	func() interface{} { return &stotypes.Volume{} },
	func(record interface{}) []byte {
		return volumeIntIdToBytes(record.(*stotypes.Volume).ID)
	})

var VolumeMountRepository = blorm.NewSimpleRepo(
	"volumemounts",
	func() interface{} { return &stotypes.VolumeMount{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.VolumeMount).ID) })

var DirectoryRepository = blorm.NewSimpleRepo(
	"directories",
	func() interface{} { return &stotypes.Directory{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Directory).ID) })

var SubdirectoriesIndex = blorm.NewValueIndex("parent", DirectoryRepository, func(record interface{}, index func(val []byte)) {
	dir := record.(*stotypes.Directory)

	if dir.Parent != "" {
		index([]byte(dir.Parent))
	}
})

var CollectionRepository = blorm.NewSimpleRepo(
	"collections",
	func() interface{} { return &stotypes.Collection{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.Collection).ID) })

var CollectionsByDirectoryIndex = blorm.NewValueIndex("directory", CollectionRepository, func(record interface{}, index func(val []byte)) {
	coll := record.(*stotypes.Collection)

	index([]byte(coll.Directory))
})

var IntegrityVerificationJobRepository = blorm.NewSimpleRepo(
	"ivjobs",
	func() interface{} { return &stotypes.IntegrityVerificationJob{} },
	func(record interface{}) []byte { return []byte(record.(*stotypes.IntegrityVerificationJob).ID) })

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

// key is heading in export file under which all JSON records are dumped
var RepoByRecordType = map[string]blorm.Repository{
	"Blob":                     BlobRepository,
	"Client":                   ClientRepository,
	"Collection":               CollectionRepository,
	"Directory":                DirectoryRepository,
	"IntegrityVerificationJob": IntegrityVerificationJobRepository,
	"Node":                     NodeRepository,
	"ReplicationPolicy":        ReplicationPolicyRepository,
	"Volume":                   VolumeRepository,
	"VolumeMount":              VolumeMountRepository,
}
