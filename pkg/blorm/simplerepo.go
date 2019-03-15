package blorm

import (
	"errors"
	"github.com/asdine/storm/codec/msgpack"
	"go.etcd.io/bbolt"
)

type simpleRepository struct {
	bucketName  []byte
	alloc       func() interface{}
	idExtractor func(record interface{}) []byte
}

func NewSimpleRepo(
	bucketName string,
	alloc func() interface{},
	idExtractor func(record interface{}) []byte,
) Repository {
	return &simpleRepository{[]byte(bucketName), alloc, idExtractor}
}

func (r *simpleRepository) Alloc() interface{} {
	return r.alloc()
}

func (r *simpleRepository) OpenByPrimaryKey(id []byte, record interface{}, tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errors.New("no bucket")
	}

	data := bucket.Get(id)
	if data == nil {
		return ErrNotFound
	}

	if err := msgpack.Codec.Unmarshal(data, record); err != nil {
		return err
	}

	return nil
}

func (r *simpleRepository) Update(record interface{}, tx *bolt.Tx) error {
	bucket, err := tx.CreateBucketIfNotExists(r.bucketName)
	if err != nil {
		return err
	}

	id := r.idExtractor(record)

	data, err := msgpack.Codec.Marshal(record)
	if err != nil {
		return err
	}

	return bucket.Put(id, data)
}

func (r *simpleRepository) Delete(record interface{}, tx *bolt.Tx) error {
	bucket, err := tx.CreateBucketIfNotExists(r.bucketName)
	if err != nil {
		return err
	}

	id := r.idExtractor(record)

	if bucket.Get(id) == nil { // Delete() does not return error for non-existing keys
		return errors.New("record to delete does not exist")
	}

	return bucket.Delete(id)
}

func (r *simpleRepository) Each(fn func(record interface{}), tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errors.New("no bucket")
	}

	all := bucket.Cursor()
	for key, value := all.First(); key != nil; key, value = all.Next() {
		record := r.alloc()

		if err := msgpack.Codec.Unmarshal(value, record); err != nil {
			return err
		}

		fn(record)
	}

	return nil
}
