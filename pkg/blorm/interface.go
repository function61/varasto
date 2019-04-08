package blorm

// Bolt Light ORM, doesn't do much else than persist structs into Bolt..

import (
	"errors"
	"go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("database: record not found")

type Repository interface {
	Bootstrap(tx *bolt.Tx) error
	DefineSetIndex(name string, memberEvaluator setIndexMemberEvaluator) SetIndexApi
	OpenByPrimaryKey(id []byte, record interface{}, tx *bolt.Tx) error
	Update(record interface{}, tx *bolt.Tx) error
	Delete(record interface{}, tx *bolt.Tx) error
	Each(fn func(record interface{}) error, tx *bolt.Tx) error
	Alloc() interface{}
}
