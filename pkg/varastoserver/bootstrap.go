package varastoserver

import (
	"github.com/asdine/storm"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"log"
)

func bootstrap(db *storm.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newNode := varastotypes.Node{
		ID:   varastoutils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	recordsToSave := []interface{}{
		&newNode,
		&varastotypes.Directory{
			ID:     "root",
			Parent: "", // root doesn't have parent
			Name:   "root",
		},
		&varastotypes.ReplicationPolicy{
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
