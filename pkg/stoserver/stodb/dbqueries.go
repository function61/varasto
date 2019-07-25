package stodb

import (
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

type dbQueries struct {
	tx *bolt.Tx
}

func Read(tx *bolt.Tx) *dbQueries {
	return &dbQueries{tx}
}

func (d *dbQueries) Blob(ref stotypes.BlobRef) (*stotypes.Blob, error) {
	record := &stotypes.Blob{}
	if err := BlobRepository.OpenByPrimaryKey([]byte(ref), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) Collection(id string) (*stotypes.Collection, error) {
	record := &stotypes.Collection{}
	if err := CollectionRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) CollectionsByDirectory(dirId string) ([]stotypes.Collection, error) {
	collections := []stotypes.Collection{}

	// TODO: might need to index this later for better perf..
	if err := CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)
		if coll.Directory == dirId {
			collections = append(collections, *coll)
		}

		return nil
	}, d.tx); err != nil {
		return nil, err
	}

	return collections, nil
}

func (d *dbQueries) Directory(id string) (*stotypes.Directory, error) {
	record := &stotypes.Directory{}
	if err := DirectoryRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) SubDirectories(of string) ([]stotypes.Directory, error) {
	subDirs := []stotypes.Directory{}

	// TODO: might need to index this later for better perf..
	if err := DirectoryRepository.Each(func(record interface{}) error {
		dir := record.(*stotypes.Directory)
		if dir.Parent == of {
			subDirs = append(subDirs, *dir)
		}

		return nil
	}, d.tx); err != nil {
		return nil, err
	}

	return subDirs, nil
}

func (d *dbQueries) Volume(id int) (*stotypes.Volume, error) {
	record := &stotypes.Volume{}
	if err := VolumeRepository.OpenByPrimaryKey(volumeIntIdToBytes(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) VolumeMount(id string) (*stotypes.VolumeMount, error) {
	record := &stotypes.VolumeMount{}
	if err := VolumeMountRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) Node(id string) (*stotypes.Node, error) {
	record := &stotypes.Node{}
	if err := NodeRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) ReplicationPolicy(id string) (*stotypes.ReplicationPolicy, error) {
	record := &stotypes.ReplicationPolicy{}
	if err := ReplicationPolicyRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}
