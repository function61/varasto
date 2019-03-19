package varastoserver

import (
	"encoding/binary"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/varastotypes"
)

var BlobRepository = blorm.NewSimpleRepo(
	"blobs",
	func() interface{} { return &varastotypes.Blob{} },
	func(record interface{}) []byte { return record.(*varastotypes.Blob).Ref })

var BlobsPendingReplicationIndex = BlobRepository.DefineSetIndex("pending_replication", func(record interface{}) bool {
	return record.(*varastotypes.Blob).IsPendingReplication
})

var NodeRepository = blorm.NewSimpleRepo(
	"nodes",
	func() interface{} { return &varastotypes.Node{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.Node).ID) })

var ClientRepository = blorm.NewSimpleRepo(
	"clients",
	func() interface{} { return &varastotypes.Client{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.Client).ID) })

var ReplicationPolicyRepository = blorm.NewSimpleRepo(
	"replicationpolicies",
	func() interface{} { return &varastotypes.ReplicationPolicy{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.ReplicationPolicy).ID) })

var VolumeRepository = blorm.NewSimpleRepo(
	"volumes",
	func() interface{} { return &varastotypes.Volume{} },
	func(record interface{}) []byte {
		return volumeIntIdToBytes(record.(*varastotypes.Volume).ID)
	})

var VolumeMountRepository = blorm.NewSimpleRepo(
	"volumemounts",
	func() interface{} { return &varastotypes.VolumeMount{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.VolumeMount).ID) })

var DirectoryRepository = blorm.NewSimpleRepo(
	"directories",
	func() interface{} { return &varastotypes.Directory{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.Directory).ID) })

var CollectionRepository = blorm.NewSimpleRepo(
	"collections",
	func() interface{} { return &varastotypes.Collection{} },
	func(record interface{}) []byte { return []byte(record.(*varastotypes.Collection).ID) })

// helpers

func volumeIntIdToBytes(id int) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(id))
	return b
}

// appenders. Go surely would need some generic love..

func clientAppender(slice *[]varastotypes.Client) func(record interface{}) {
	return func(record interface{}) {
		*slice = append(*slice, *record.(*varastotypes.Client))
	}
}

func nodeAppender(slice *[]varastotypes.Node) func(record interface{}) {
	return func(record interface{}) {
		*slice = append(*slice, *record.(*varastotypes.Node))
	}
}

func replicationPolicyAppender(slice *[]varastotypes.ReplicationPolicy) func(record interface{}) {
	return func(record interface{}) {
		*slice = append(*slice, *record.(*varastotypes.ReplicationPolicy))
	}
}

func volumeAppender(slice *[]varastotypes.Volume) func(record interface{}) {
	return func(record interface{}) {
		*slice = append(*slice, *record.(*varastotypes.Volume))
	}
}

func volumeMountAppender(slice *[]varastotypes.VolumeMount) func(record interface{}) {
	return func(record interface{}) {
		*slice = append(*slice, *record.(*varastotypes.VolumeMount))
	}
}
