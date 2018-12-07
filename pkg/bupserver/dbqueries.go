package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
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

func (d *dbQueries) Blob(ref buptypes.BlobRef) (*buptypes.Blob, error) {
	record := &buptypes.Blob{}
	if err := d.tx.One("Ref", ref, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Collection(id string) (*buptypes.Collection, error) {
	record := &buptypes.Collection{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Directory(id string) (*buptypes.Directory, error) {
	record := &buptypes.Directory{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Volume(id int) (*buptypes.Volume, error) {
	record := &buptypes.Volume{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) Node(id string) (*buptypes.Node, error) {
	record := &buptypes.Node{}
	if err := d.tx.One("ID", id, record); err != nil {
		return nil, translateDbError(err)
	}

	return record, nil
}

func (d *dbQueries) ReplicationPolicy(id string) (*buptypes.ReplicationPolicy, error) {
	record := &buptypes.ReplicationPolicy{}
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
