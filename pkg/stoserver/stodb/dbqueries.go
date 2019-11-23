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

func (d *dbQueries) ScheduledJob(id string) (*stotypes.ScheduledJob, error) {
	record := &stotypes.ScheduledJob{}
	if err := ScheduledJobRepository.OpenByPrimaryKey([]byte(id), record, d.tx); err != nil {
		return nil, err
	}

	return record, nil
}

func (d *dbQueries) CollectionsByDirectory(dirId string) ([]stotypes.Collection, error) {
	collections := []stotypes.Collection{}

	return collections, CollectionsByDirectoryIndex.Query([]byte(dirId), StartFromFirst, func(id []byte) error {
		coll, err := d.Collection(string(id))
		if err != nil {
			return err
		}

		collections = append(collections, *coll)

		return nil
	}, d.tx)
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

	return subDirs, SubdirectoriesIndex.Query([]byte(of), StartFromFirst, func(id []byte) error {
		dir, err := d.Directory(string(id))
		if err != nil {
			return err
		}

		subDirs = append(subDirs, *dir)

		return nil
	}, d.tx)
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
