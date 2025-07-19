// "Bolt Light ORM", doesn't do much else than persist structs into Bolt..
// this was born because: https://github.com/asdine/storm/issues/222#issuecomment-472791001
//
// Warning: don't Each() and Delete() at the same time. Deletion messes with the iteration
//
//	order somehow, and I observed half of the records was deleted when I tried to delete all.
package blorm

import (
	"errors"

	"go.etcd.io/bbolt"
)

var (
	ErrNotFound       = errors.New("database: record not found")
	ErrBucketNotFound = errors.New("bucket not found")
	ErrStopIteration  = errors.New("blorm: stop iteration")
)

type Repository interface {
	Bootstrap(tx *bbolt.Tx) error
	// returns ErrNotFound if record not found
	// returns ErrBucketNotFound if bootstrap not done for bucket
	OpenByPrimaryKey(id []byte, record any, tx *bbolt.Tx) error
	Update(record any, tx *bbolt.Tx) error
	Delete(record any, tx *bbolt.Tx) error
	// return blorm.StopIteration from "fn" to stop iteration. that error is not returned
	// to the API caller
	Each(fn func(record any) error, tx *bbolt.Tx) error
	// rules of Each() also apply here
	EachFrom(from []byte, fn func(record any) error, tx *bbolt.Tx) error
	Alloc() any
}
