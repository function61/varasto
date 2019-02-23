package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/gokit/logex"
	"log"
)

func bootstrap(db *storm.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newNode := buptypes.Node{
		ID:   buputils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	recordsToSave := []interface{}{
		&newNode,
		&buptypes.Directory{
			ID:     "root",
			Parent: "", // root doesn't have parent
			Name:   "root",
		},
		&buptypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{1, 2}, // FIXME: this assumes 1 and 2 will be created soon..
		},
	}

	for _, recordToSave := range recordsToSave {
		if err := tx.Save(recordToSave); err != nil {
			return err
		}
	}

	if err := tx.Set("settings", "nodeId", &newNode.ID); err != nil {
		return err
	}

	return tx.Commit()
}
