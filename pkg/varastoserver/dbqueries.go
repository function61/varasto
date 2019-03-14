package varastoserver

import (
	"github.com/asdine/storm"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/pkg/errors"
)

// abstraction for storm's storm.ErrNotFound
var ErrDbRecordNotFound = errors.New("database: record not found")

type dbQueries struct {
	tx storm.Node
}

func QueryWithTx(tx storm.Node) *dbQueries {
	return &dbQueries{tx}
}

func (d *dbQueries) Blob(ref varastotypes.BlobRef) (*varastotypes.Blob, error) {
	record := &varastotypes.Blob{}
	if err := d.tx.One("Ref", ref, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Collection(id string) (*varastotypes.Collection, error) {
	record := &varastotypes.Collection{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) CollectionsByDirectory(dirId string) ([]varastotypes.Collection, error) {
	collections := []varastotypes.Collection{}
	if err := d.tx.Find("Directory", dirId, &collections); err != nil && err != storm.ErrNotFound {
		return nil, translateDbError(err)
	}

	return collections, nil
}

func (d *dbQueries) Directory(id string) (*varastotypes.Directory, error) {
	record := &varastotypes.Directory{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) SubDirectories(of string) ([]varastotypes.Directory, error) {
	subDirs := []varastotypes.Directory{}
	if err := d.tx.Find("Parent", of, &subDirs); err != nil && err != storm.ErrNotFound {
		return nil, err
	}

	return subDirs, nil
}

func (d *dbQueries) Volume(id int) (*varastotypes.Volume, error) {
	record := &varastotypes.Volume{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) VolumeMount(id string) (*varastotypes.VolumeMount, error) {
	record := &varastotypes.VolumeMount{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Node(id string) (*varastotypes.Node, error) {
	record := &varastotypes.Node{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) ReplicationPolicy(id string) (*varastotypes.ReplicationPolicy, error) {
	record := &varastotypes.ReplicationPolicy{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func translateDbError(err error) error {
	if err == storm.ErrNotFound {
		return ErrDbRecordNotFound
	} else {
		return errors.Wrap(err, "database")
	}
}
