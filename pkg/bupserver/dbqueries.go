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
	blob := &buptypes.Blob{}
	if err := d.tx.One("Ref", ref, blob); err != nil {
		return nil, translateDbError(err)
	}

	return blob, nil
}

func (d *dbQueries) Collection(id string) (*buptypes.Collection, error) {
	coll := &buptypes.Collection{}
	if err := d.tx.One("ID", id, coll); err != nil {
		return nil, translateDbError(err)
	}

	return coll, nil
}

func translateDbError(err error) error {
	if err == storm.ErrNotFound {
		return ErrDbRecordNotFound
	} else {
		return errors.Wrap(err, "database")
	}
}
