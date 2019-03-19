package blorm

import (
	"go.etcd.io/bbolt"
)

type SetIndexApi interface {
	Bucket(tx *bolt.Tx) *bolt.Bucket
}

type setIndexMemberEvaluator func(record interface{}) bool

var dropFromIndex setIndexMemberEvaluator = func(interface{}) bool { return false }

// implements a set-based index
type index struct {
	indexBucketName []byte                  // <reponame>:<indexName>
	memberEvaluator setIndexMemberEvaluator // checks if record should be member of the set
}

func (i *index) Bucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(i.indexBucketName)
}

func (i *index) act(record interface{}, repo *simpleRepository, tx *bolt.Tx) error {
	return i.actWithEvaluator(record, repo, tx, i.memberEvaluator)
}

func (i *index) actWithEvaluator(
	record interface{},
	repo *simpleRepository,
	tx *bolt.Tx,
	memberEvaluator setIndexMemberEvaluator,
) error {
	indexBucket := tx.Bucket(i.indexBucketName)

	id := repo.idExtractor(record)

	if memberEvaluator(record) {
		return indexBucket.Put(id, nil)
	} else {
		return indexBucket.Delete(id)
	}
}
