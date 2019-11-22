package stodb

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
	"log"
)

// opens BoltDB database
func Open(dbLocation string) (*bolt.DB, error) {
	return bolt.Open(dbLocation, 0700, nil)
}

func Bootstrap(db *bolt.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	// be extra safe and scan the DB to see that it is totally empty
	if err := tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
		return fmt.Errorf("DB not empty, found bucket: %s", name)
	}); err != nil {
		return err
	}

	if err := BootstrapRepos(tx); err != nil {
		return err
	}

	newNode := &stotypes.Node{
		ID:   stoutils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	results := []error{
		NodeRepository.Update(newNode, tx),
		DirectoryRepository.Update(stotypes.NewDirectory(
			"root",
			"",
			"root",
			string(stoservertypes.DirectoryTypeGeneric)), tx),
		ReplicationPolicyRepository.Update(&stotypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{},
		}, tx),
		CfgNodeId.Set(newNode.ID, tx),
	}

	if err := allOk(results); err != nil {
		return err
	}

	return tx.Commit()
}

func BootstrapRepos(tx *bolt.Tx) error {
	for _, repo := range RepoByRecordType {
		if err := repo.Bootstrap(tx); err != nil {
			return err
		}
	}

	return nil
}

func allOk(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func ignoreError(err error) {
	// no-op
}
