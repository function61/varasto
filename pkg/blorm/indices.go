package blorm

import (
	"bytes"

	"go.etcd.io/bbolt"
)

/*	types of indices
	================

	setIndex
	--------
	(pending_replication, "_", id) = nil

	simpleIndex
	-----------
	(by_parent, parentId, id) = nil
*/

var (
	StartFromFirst = []byte("")
)

// fully qualified index reference, including the index name
type qualifiedIndexRef struct {
	indexName string // looks like directories:by_parent
	val       []byte // for setIndex this is always " "
	id        []byte // primary key of record the index entry refers to
}

func (i *qualifiedIndexRef) Equals(other *qualifiedIndexRef) bool {
	return i.indexName == other.indexName &&
		bytes.Equal(i.val, other.val) &&
		bytes.Equal(i.id, other.id)
}

func mkIndexRef(indexName string, val []byte, id []byte) qualifiedIndexRef {
	return qualifiedIndexRef{indexName, val, id}
}

type Index interface {
	// only for our internal use
	extractIndexRefs(record interface{}) []qualifiedIndexRef
}

type setIndexApi interface {
	// return StopIteration if you want to stop mid-iteration (nil error will be returned by Query() )
	Query(start []byte, fn func(id []byte) error, tx *bbolt.Tx) error
	Index
}

type byValueIndexApi interface {
	// return StopIteration if you want to stop mid-iteration (nil error will be returned by Query() )
	Query(val []byte, start []byte, fn func(id []byte) error, tx *bbolt.Tx) error
	Index
}

type setIndex struct {
	repo            *SimpleRepository
	name            string // looks like <repoBucketName>:<indexName>
	memberEvaluator func(record interface{}) bool
}

func (s *setIndex) extractIndexRefs(record interface{}) []qualifiedIndexRef {
	if s.memberEvaluator(record) {
		return []qualifiedIndexRef{
			mkIndexRef(s.name, []byte(" "), s.repo.idExtractor(record)),
		}
	}

	return []qualifiedIndexRef{}
}

func (s *setIndex) Query(start []byte, fn func(id []byte) error, tx *bbolt.Tx) error {
	// " " is required because empty key is not supported
	return indexQueryShared(s.name, []byte(" "), start, fn, tx)
}

func NewSetIndex(name string, repo *SimpleRepository, memberEvaluator func(record interface{}) bool) setIndexApi {
	idx := &setIndex{repo, string(repo.bucketName) + ":" + name, memberEvaluator}

	repo.indices = append(repo.indices, idx)

	return idx
}

type byValueIndex struct {
	repo            *SimpleRepository
	name            string // looks like <repoBucketName>:<indexName>
	memberEvaluator func(record interface{}, push func(val []byte))
}

func (b *byValueIndex) extractIndexRefs(record interface{}) []qualifiedIndexRef {
	qualifiedRefs := []qualifiedIndexRef{}
	b.memberEvaluator(record, func(val []byte) {
		if len(val) == 0 {
			panic("cannot index by empty value")
		}
		qualifiedRefs = append(qualifiedRefs, mkIndexRef(b.name, val, b.repo.idExtractor(record)))
	})

	return qualifiedRefs
}

func (b *byValueIndex) Query(value []byte, start []byte, fn func(id []byte) error, tx *bbolt.Tx) error {
	return indexQueryShared(b.name, value, start, fn, tx)
}

// used both by byValueIndex and by setIndex
func indexQueryShared(indexName string, value []byte, start []byte, fn func(id []byte) error, tx *bbolt.Tx) error {
	// the nil part is not used by indexBucketRefForQuery()
	bucket := indexBucketRefForQuery(mkIndexRef(indexName, value, nil), tx)
	if bucket == nil { // index doesn't exist => not matching entries
		return nil
	}

	idx := bucket.Cursor()

	var key []byte
	if bytes.Equal(start, StartFromFirst) {
		key, _ = idx.First()
	} else {
		key, _ = idx.Seek(start)
	}

	for ; key != nil; key, _ = idx.Next() {
		if err := fn(makeCopy(key)); err != nil {
			if err == StopIteration {
				return nil
			} else {
				return err
			}
		}
	}

	return nil
}

func NewValueIndex(name string, repo *SimpleRepository, memberEvaluator func(record interface{}, push func(val []byte))) byValueIndexApi {
	idx := &byValueIndex{repo, string(repo.bucketName) + ":" + name, memberEvaluator}

	repo.indices = append(repo.indices, idx)

	return idx
}

func indexRefExistsIn(ir qualifiedIndexRef, coll []qualifiedIndexRef) bool {
	for _, other := range coll {
		other := other // pin
		if ir.Equals(&other) {
			return true
		}
	}

	return false
}

func indexBucketRefForQuery(ref qualifiedIndexRef, tx *bbolt.Tx) *bbolt.Bucket {
	// directories:by_parent
	lvl1 := tx.Bucket([]byte(ref.indexName))
	if lvl1 == nil {
		return nil
	}

	return lvl1.Bucket(ref.val)
}

func indexBucketRefForWrite(ref qualifiedIndexRef, tx *bbolt.Tx) *bbolt.Bucket {
	// directories:by_parent
	lvl1, err := tx.CreateBucketIfNotExists([]byte(ref.indexName))
	if err != nil {
		panic(err)
	}

	lvl2, err := lvl1.CreateBucketIfNotExists(ref.val)
	if err != nil {
		panic(err)
	}

	return lvl2
}

// https://github.com/boltdb/bolt/issues/658#issuecomment-277898467
func makeCopy(from []byte) []byte {
	copied := make([]byte, len(from))
	copy(copied, from)
	return copied
}
