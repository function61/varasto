// "Bolt Light ORM", doesn't do much else than persist structs into Bolt..
package blorm

import (
	"errors"
	"go.etcd.io/bbolt"
)

var (
	ErrNotFound   = errors.New("database: record not found")
	StopIteration = errors.New("blorm: stop iteration")
)

type Repository interface {
	Bootstrap(tx *bolt.Tx) error
	DefineSetIndex(name string, memberEvaluator setIndexMemberEvaluator) SetIndexApi
	OpenByPrimaryKey(id []byte, record interface{}, tx *bolt.Tx) error
	Update(record interface{}, tx *bolt.Tx) error
	Delete(record interface{}, tx *bolt.Tx) error
	// return blorm.StopIteration from "fn" to stop iteration. that error is not returned
	// to the API caller
	Each(fn func(record interface{}) error, tx *bolt.Tx) error
	// rules of Each() also apply here
	EachFrom(from []byte, fn func(record interface{}) error, tx *bolt.Tx) error
	Alloc() interface{}
}
