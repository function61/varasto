package blorm

import (
	"errors"
	"fmt"

	"github.com/asdine/storm/codec/msgpack"
	"go.etcd.io/bbolt"
)

type SimpleRepository struct {
	bucketName  []byte
	alloc       func() interface{}
	idExtractor func(record interface{}) []byte
	indices     []Index
}

func NewSimpleRepo(bucketName string, allocator func() interface{}, idExtractor func(interface{}) []byte) *SimpleRepository {
	return &SimpleRepository{
		bucketName:  []byte(bucketName),
		alloc:       allocator,
		idExtractor: idExtractor,
		indices:     []Index{},
	}
}

func (r *SimpleRepository) Bootstrap(tx *bbolt.Tx) error {
	_, err := tx.CreateBucket(r.bucketName)
	return err
}

func (r *SimpleRepository) Alloc() interface{} {
	return r.alloc()
}

func (r *SimpleRepository) OpenByPrimaryKey(id []byte, record interface{}, tx *bbolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return ErrBucketNotFound
	}

	data := bucket.Get(id)
	if data == nil {
		return ErrNotFound
	}

	if err := msgpack.Codec.Unmarshal(data, record); err != nil {
		return fmt.Errorf("repo[%s] record[%s]: %v", r.bucketName, id, err)
	}

	return nil
}

func (r *SimpleRepository) Update(record interface{}, tx *bbolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return ErrBucketNotFound
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

func (r *SimpleRepository) Delete(record interface{}, tx *bbolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return ErrBucketNotFound
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

func (r *SimpleRepository) Each(fn func(record interface{}) error, tx *bbolt.Tx) error {
	return r.EachFrom([]byte(""), fn, tx)
}

func (r *SimpleRepository) EachFrom(from []byte, fn func(record interface{}) error, tx *bbolt.Tx) error {
	bucket := tx.Bucket(r.bucketName)
	if bucket == nil {
		return ErrBucketNotFound
	}

	all := bucket.Cursor()
	for key, value := all.Seek(from); key != nil; key, value = all.Next() {
		record := r.alloc()

		if err := msgpack.Codec.Unmarshal(value, record); err != nil {
			return fmt.Errorf("repo[%s] record[%s]: %v", r.bucketName, key, err)
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

func (r *SimpleRepository) indexRefsForRecord(record interface{}) []qualifiedIndexRef {
	refs := []qualifiedIndexRef{}

	for _, repoIndex := range r.indices {
		refs = append(refs, repoIndex.extractIndexRefs(record)...)
	}

	return refs
}

func (r *SimpleRepository) updateIndices(
	oldIndices []qualifiedIndexRef,
	newIndices []qualifiedIndexRef,
	tx *bbolt.Tx,
) error {
	for _, old := range oldIndices {
		// old doesn't exist in new => drop
		if !indexRefExistsIn(old, newIndices) {
			if err := old.Drop(tx); err != nil {
				return err
			}
		}
	}

	for _, nu := range newIndices {
		// new doesn't exist in old => add
		if !indexRefExistsIn(nu, oldIndices) {
			if err := nu.Write(tx); err != nil {
				return err
			}
		}
	}

	return nil
}
