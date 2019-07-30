package stodb

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
	"log"
)

var (
	configBucketKey = []byte("config")
	CfgNodeId       = ConfigAccessor("nodeId")
)

// opens BoltDB database
func Open(dbLocation string) (*bolt.DB, error) {
	return bolt.Open(dbLocation, 0700, nil)
}

func Bootstrap(db *bolt.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	if err := BootstrapRepos(db); err != nil {
		return err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newNode := &stotypes.Node{
		ID:   stoutils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	results := []error{
		NodeRepository.Update(newNode, tx),
		DirectoryRepository.Update(stotypes.NewDirectory("root", "", "root"), tx),
		ReplicationPolicyRepository.Update(&stotypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{1, 2}, // FIXME: this assumes 1 and 2 will be created soon..
		}, tx),
		BootstrapSetNodeId(newNode.ID, tx),
	}

	if err := allOk(results); err != nil {
		return err
	}

	return tx.Commit()
}

func BootstrapSetNodeId(nodeId string, tx *bolt.Tx) error {
	// errors if already exists (this is important)
	configBucket, err := tx.CreateBucket(configBucketKey)
	if err != nil {
		return err
	}

	return configBucket.Put(CfgNodeId.key, []byte(nodeId))
	// TODO: use (but verify that tx.Bucket(foo) after CreateBucket(foo) works)
	// return CfgNodeId.Set(nodeId, tx)
}

func BootstrapRepos(db *bolt.DB) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, repo := range RepoByRecordType {
		if err := repo.Bootstrap(tx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func allOk(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
