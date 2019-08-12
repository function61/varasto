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
	indices     []Index
}

func NewSimpleRepo(bucketName string, allocator func() interface{}, idExtractor func(interface{}) []byte) *simpleRepository {
	return &simpleRepository{
		bucketName:  []byte(bucketName),
		alloc:       allocator,
		idExtractor: idExtractor,
		indices:     []Index{},
	}
}

func (r *simpleRepository) Bootstrap(tx *bolt.Tx) error {
	_, err := tx.CreateBucket(r.bucketName)
	return err
}

func (r *simpleRepository) Alloc() interface{} {
	return r.alloc()
}

func (r *simpleRepository) OpenByPrimaryKey(id []byte, record interface{}, tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errNoBucket
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
		return errNoBucket
	}

	id := r.idExtractor(record)

	data, err := msgpack.Codec.Marshal(record)
	if err != nil {
		return err
	}

	oldImage := r.alloc()

	errOpenOld := r.OpenByPrimaryKey(id, oldImage, tx)
	if errOpenOld != nil && errOpenOld != ErrNotFound {
		return errOpenOld
	}

	oldIndices := []qualifiedIndexRef{}
	newIndices := r.indexRefsForRecord(record)

	if errOpenOld != ErrNotFound { // we have old and new image, must compare indices for old and new image
		oldIndices = r.indexRefsForRecord(oldImage)
	}

	if err := r.updateIndices(oldIndices, newIndices, tx); err != nil {
		return err
	}

	return bucket.Put(id, data)
}

func (r *simpleRepository) Delete(record interface{}, tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errNoBucket
	}

	id := r.idExtractor(record)

	if bucket.Get(id) == nil { // bucket.Delete() does not return error for non-existing keys
		return errors.New("record to delete does not exist")
	}

	oldIndices := r.indexRefsForRecord(record)
	newIndices := []qualifiedIndexRef{} // = drop

	if err := r.updateIndices(oldIndices, newIndices, tx); err != nil {
		return err
	}

	return bucket.Delete(id)
}

func (r *simpleRepository) Each(fn func(record interface{}) error, tx *bolt.Tx) error {
	return r.EachFrom([]byte(""), fn, tx)
}

func (r *simpleRepository) EachFrom(from []byte, fn func(record interface{}) error, tx *bolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return errNoBucket
	}

	all := bucket.Cursor()
	for key, value := all.Seek(from); key != nil; key, value = all.Next() {
		record := r.alloc()

		if err := msgpack.Codec.Unmarshal(value, record); err != nil {
			return err
		}

		if err := fn(record); err != nil {
			if err == StopIteration {
				return nil // not an error, so don't give one out
			}

			return err
		}
	}

	return nil
}

func (r *simpleRepository) indexRefsForRecord(record interface{}) []qualifiedIndexRef {
	refs := []qualifiedIndexRef{}

	for _, repoIndex := range r.indices {
		refs = append(refs, repoIndex.extractIndexRefs(record)...)
	}

	return refs
}

func (r *simpleRepository) updateIndices(oldIndices []qualifiedIndexRef, newIndices []qualifiedIndexRef, tx *bolt.Tx) error {
	for _, old := range oldIndices {
		if !indexRefExistsIn(old, newIndices) {
			if err := indexBucketRefForWrite(old, tx).Delete(old.valAndId.id); err != nil {
				return err
			}
		}
	}

	for _, nu := range newIndices {
		if !indexRefExistsIn(nu, oldIndices) {
			if err := indexBucketRefForWrite(nu, tx).Put(nu.valAndId.id, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
