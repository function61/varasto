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
	indices     []index
}

func NewSimpleRepo(
	bucketName string,
	alloc func() interface{},
	idExtractor func(record interface{}) []byte,
) Repository {
	return &simpleRepository{[]byte(bucketName), alloc, idExtractor, []index{}}
}

func (r *simpleRepository) DefineSetIndex(name string, memberEvaluator setIndexMemberEvaluator) SetIndexApi {
	idx := index{
		indexBucketName: []byte(string(r.bucketName) + ":" + name),
		memberEvaluator: memberEvaluator,
	}

	r.indices = append(r.indices, idx)

	return &idx
}

func (r *simpleRepository) Bootstrap(tx *bolt.Tx) error {
	if _, err := tx.CreateBucket(r.bucketName); err != nil {
		return err
	}

	for _, index := range r.indices {
		if _, err := tx.CreateBucket(index.indexBucketName); err != nil {
			return err
		}
	}

	return nil
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
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errors.New("no bucket")
	}

	id := r.idExtractor(record)

	data, err := msgpack.Codec.Marshal(record)
	if err != nil {
		return err
	}

	if err := bucket.Put(id, data); err != nil {
		return err
	}

	for _, idx := range r.indices {
		if err := idx.act(record, r, tx); err != nil {
			return err
		}
	}

	return nil
}

func (r *simpleRepository) Delete(record interface{}, tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errors.New("no bucket")
	}

	id := r.idExtractor(record)

	if bucket.Get(id) == nil { // bucket.Delete() does not return error for non-existing keys
		return errors.New("record to delete does not exist")
	}

	for _, index := range r.indices {
		if err := index.actWithEvaluator(record, r, tx, dropFromIndex); err != nil {
			return err
		}
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
