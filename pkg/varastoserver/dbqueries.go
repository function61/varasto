package varastoserver

import (
	"github.com/function61/varasto/pkg/varastotypes"
	"go.etcd.io/bbolt"
)

type dbQueries struct {
	tx *bolt.Tx
}

func QueryWithTx(tx *bolt.Tx) *dbQueries {
	return &dbQueries{tx}
}

func (d *dbQueries) Blob(ref varastotypes.BlobRef) (*varastotypes.Blob, error) {
	record := &varastotypes.Blob{}
	if err := BlobRepository.OpenByPrimaryKey([]byte(ref), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) Collection(id string) (*varastotypes.Collection, error) {
	record := &varastotypes.Collection{}
	if err := CollectionRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) CollectionsByDirectory(dirId string) ([]varastotypes.Collection, error) {
	collections := []varastotypes.Collection{}

	// TODO: might need to index this later for better perf..
	if err := CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*varastotypes.Collection)
		if coll.Directory == dirId {
			collections = append(collections, *coll)
		}

		return nil
	}, d.tx); err != nil {
		return nil, err
	}

	return collections, nil
}

func (d *dbQueries) Directory(id string) (*varastotypes.Directory, error) {
	record := &varastotypes.Directory{}
	if err := DirectoryRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) SubDirectories(of string) ([]varastotypes.Directory, error) {
	subDirs := []varastotypes.Directory{}

	// TODO: might need to index this later for better perf..
	if err := DirectoryRepository.Each(func(record interface{}) error {
		dir := record.(*varastotypes.Directory)
		if dir.Parent == of {
			subDirs = append(subDirs, *dir)
		}

		return nil
	}, d.tx); err != nil {
		return nil, err
	}

	return subDirs, nil
}

func (d *dbQueries) Volume(id int) (*varastotypes.Volume, error) {
	record := &varastotypes.Volume{}
	if err := VolumeRepository.OpenByPrimaryKey(volumeIntIdToBytes(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) VolumeMount(id string) (*varastotypes.VolumeMount, error) {
	record := &varastotypes.VolumeMount{}
	if err := VolumeMountRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) Node(id string) (*varastotypes.Node, error) {
	record := &varastotypes.Node{}
	if err := NodeRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) ReplicationPolicy(id string) (*varastotypes.ReplicationPolicy, error) {
	record := &varastotypes.ReplicationPolicy{}
	if err := ReplicationPolicyRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}
