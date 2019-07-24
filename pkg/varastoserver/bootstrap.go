package varastoserver

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/varastoserver/stodb"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"go.etcd.io/bbolt"
	"log"
)

var (
	configBucketKey     = []byte("config")
	configBucketNodeKey = []byte("nodeId")
)

func bootstrap(db *bolt.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	if err := bootstrapRepos(db); err != nil {
		return err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newNode := &varastotypes.Node{
		ID:   varastoutils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	results := []error{
		stodb.NodeRepository.Update(newNode, tx),
		stodb.DirectoryRepository.Update(varastotypes.NewDirectory("root", "", "root"), tx),
		stodb.ReplicationPolicyRepository.Update(&varastotypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{1, 2}, // FIXME: this assumes 1 and 2 will be created soon..
		}, tx),
		bootstrapSetNodeId(newNode.ID, tx),
	}

	if err := allOk(results); err != nil {
		return err
	}

	return tx.Commit()
}

func bootstrapSetNodeId(nodeId string, tx *bolt.Tx) error {
	// errors if already exists
	configBucket, err := tx.CreateBucket(configBucketKey)
	if err != nil {
		return err
	}

	return configBucket.Put(configBucketNodeKey, []byte(nodeId))
}

func bootstrapRepos(db *bolt.DB) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, repo := range repoByRecordType {
		if err := repo.Bootstrap(tx); err != nil {
			return err
		}
	}

	return tx.Commit()
}
