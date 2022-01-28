package blorm

import (
	"bytes"

	"go.etcd.io/bbolt"
)

/*	types of indices
	================

	setIndex (example: pending_replication)
	--------
	("_", id) = nil


	valueIndex (example: by_parent)
	-----------
	(parentId, id) = nil


	rangeIndex (example: change_timestamp)
	----------
	(timestamp) = id

	TODO: setIndex could now be refactored to not utilize the dummy partition, but we're
	      not doing that right now to prevent breaking old data
*/

var (
	StartFromFirst = []byte("")
)

type Index interface {
	// only for our internal use
	extractIndexRefs(record interface{}) []qualifiedIndexRef
}

// fully qualified index reference, including the index name
type qualifiedIndexRef struct {
	indexName []byte // looks like directories:by_parent
	partition []byte // for setIndex this is always " "
	sortKey   []byte // primary key of record the index entry refers to
	value     []byte
}

func (i *qualifiedIndexRef) Equals(other *qualifiedIndexRef) bool {
	return bytes.Equal(i.indexName, other.indexName) &&
		bytes.Equal(i.partition, other.partition) &&
		bytes.Equal(i.sortKey, other.sortKey) &&
		bytes.Equal(i.value, other.value)
}

// write index entry to DB
func (i *qualifiedIndexRef) Write(tx *bbolt.Tx) error {
	return indexBucketRefForWrite(i, tx).Put(i.sortKey, i.value)
}

// drop index entry from DB
func (i *qualifiedIndexRef) Drop(tx *bbolt.Tx) error {
	return indexBucketRefForWrite(i, tx).Delete(i.sortKey)
}

func indexBucketRefForWrite(ref *qualifiedIndexRef, tx *bbolt.Tx) *bbolt.Bucket {
	// directories:by_parent
	indexBucket, err := tx.CreateBucketIfNotExists(ref.indexName)
	if err != nil {
		panic(err)
	}

	if len(ref.partition) == 0 { // no separate partition
		return indexBucket
	}

	partitionBucket, err := indexBucket.CreateBucketIfNotExists(ref.partition)
	if err != nil {
		panic(err)
	}

	return partitionBucket
}

func mkIndexRef(indexName []byte, partition []byte, sortKey []byte, value []byte) qualifiedIndexRef {
	return qualifiedIndexRef{indexName, partition, sortKey, value}
}

type setIndexApi interface {
	// return StopIteration if you want to stop mid-iteration (nil error will be returned by Query() )
	Query(start []byte, fn func(sortKey []byte) error, tx *bbolt.Tx) error
	Index
}

type byValueIndexApi interface {
	// return StopIteration if you want to stop mid-iteration (nil error will be returned by Query() )
	Query(partition []byte, start []byte, fn func(sortKey []byte) error, tx *bbolt.Tx) error
	Index
}

type setIndex struct {
	repo            *SimpleRepository
	indexName       []byte // looks like <repoBucketName>:<indexName>
	memberEvaluator func(record interface{}) bool
}

func (s *setIndex) extractIndexRefs(record interface{}) []qualifiedIndexRef {
	if s.memberEvaluator(record) {
		return []qualifiedIndexRef{
			mkIndexRef(s.indexName, []byte(" "), s.repo.idExtractor(record), nil),
		}
	}

	return []qualifiedIndexRef{}
}

func (s *setIndex) Query(start []byte, fn func(sortKey []byte) error, tx *bbolt.Tx) error {
	// " " is required because empty bucket name is not supported
	return indexQueryShared(s.indexName, []byte(" "), start, ignoreVal(fn), tx)
}

func NewSetIndex(name string, repo *SimpleRepository, memberEvaluator func(record interface{}) bool) setIndexApi {
	idx := &setIndex{repo, mkIndexName(name, repo), memberEvaluator}

	repo.indices = append(repo.indices, idx)

	return idx
}

type byValueIndex struct {
	repo            *SimpleRepository
	indexName       []byte // looks like <repoBucketName>:<indexName>
	memberEvaluator func(record interface{}, push func(partition []byte))
}

func (b *byValueIndex) extractIndexRefs(record interface{}) []qualifiedIndexRef {
	qualifiedRefs := []qualifiedIndexRef{}
	b.memberEvaluator(record, func(partition []byte) {
		if len(partition) == 0 {
			panic("cannot index by empty value")
		}
		ref := mkIndexRef(b.indexName, partition, b.repo.idExtractor(record), nil)
		qualifiedRefs = append(qualifiedRefs, ref)
	})

	return qualifiedRefs
}

func (b *byValueIndex) Query(partition []byte, start []byte, fn func(sortKey []byte) error, tx *bbolt.Tx) error {
	return indexQueryShared(b.indexName, partition, start, ignoreVal(fn), tx)
}

// used for indices which have *nil* as the item's value
func ignoreVal(fn func(sortKey []byte) error) func(sortKey []byte, val []byte) error {
	return func(sortKey []byte, val []byte) error {
		return fn(sortKey)
	}
}

// used both by byValueIndex and by setIndex
func indexQueryShared(
	indexName []byte,
	partition []byte,
	sortKeyStartInclusive []byte,
	fn func(sortKey []byte, val []byte) error,
	tx *bbolt.Tx,
) error {
	// directories:by_parent
	indexBucket := tx.Bucket(indexName)
	if indexBucket == nil {
		return nil // index doesn't exist => no matching entries
	}

	bucketToScan := indexBucket
	if len(partition) > 0 {
		partitionBucket := indexBucket.Bucket(partition)
		if partitionBucket == nil {
			return nil // partition bucket doesn't exist => no matching entries
		}

		bucketToScan = partitionBucket
	}

	idx := bucketToScan.Cursor()

	var sortKey []byte
	var value []byte
	if bytes.Equal(sortKeyStartInclusive, StartFromFirst) {
		sortKey, value = idx.First()
	} else {
		sortKey, value = idx.Seek(sortKeyStartInclusive)
	}

	for ; sortKey != nil; sortKey, value = idx.Next() {
		if err := fn(makeCopy(sortKey), makeCopy(value)); err != nil {
			if err == StopIteration {
				return nil
			} else {
				return err
			}
		}
	}

	return nil
}

func NewValueIndex(name string, repo *SimpleRepository, memberEvaluator func(record interface{}, push func(partition []byte))) byValueIndexApi {
	idx := &byValueIndex{repo, mkIndexName(name, repo), memberEvaluator}

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

// https://github.com/boltdb/bolt/issues/658#issuecomment-277898467
func makeCopy(from []byte) []byte {
	copied := make([]byte, len(from))
	copy(copied, from)
	return copied
}

type rangeIndexApi interface {
	Index
	Query(start []byte, fn func(sortKey []byte, value []byte) error, tx *bbolt.Tx) error
}

type rangeIndex struct {
	repo            *SimpleRepository
	indexName       []byte
	memberEvaluator func(record interface{}, index func(sortKey []byte))
}

func (r *rangeIndex) Query(start []byte, fn func(sortKey []byte, value []byte) error, tx *bbolt.Tx) error {
	return indexQueryShared(r.indexName, nil, start, fn, tx)
}

func (r *rangeIndex) extractIndexRefs(record interface{}) []qualifiedIndexRef {
	refs := []qualifiedIndexRef{}

	r.memberEvaluator(record, func(sortKey []byte) {
		id := r.repo.idExtractor(record)
		refs = append(refs, mkIndexRef(r.indexName, nil, sortKey, id))
	})

	return refs
}

func NewRangeIndex(name string, repo *SimpleRepository, memberEvaluator func(record interface{}, index func(sortKey []byte))) rangeIndexApi {
	idx := &rangeIndex{repo, mkIndexName(name, repo), memberEvaluator}

	repo.indices = append(repo.indices, idx)

	return idx
}

func mkIndexName(name string, repo *SimpleRepository) []byte {
	return []byte(string(repo.bucketName) + ":" + name)
}
