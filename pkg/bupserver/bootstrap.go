package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/gokit/cryptorandombytes"
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

	volume1 := buptypes.Volume{
		ID:         1,
		Identifier: "8gxL",
		Label:      "dev vol. 1",
		Quota:      int64(1024 * 1024 * 30),
	}

	volume2 := buptypes.Volume{
		ID:         2,
		Identifier: "irG8",
		Label:      "dev vol. 2",
		Quota:      int64(1024 * 1024 * 30),
	}

	newNode := buptypes.Node{
		ID:   buputils.NewNodeId(),
		Addr: "localhost:8066",
		Name: "dev",
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	recordsToSave := []interface{}{
		&newNode,
		&buptypes.Client{
			ID:        buputils.NewClientId(),
			Name:      "Vagrant VM",
			AuthToken: cryptorandombytes.Base64Url(32),
		},
		&volume1,
		&volume2,
		&buptypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{volume1.ID, volume2.ID},
		},
		&buptypes.VolumeMount{
			ID:         buputils.NewVolumeMountId(),
			Volume:     volume1.ID,
			Node:       newNode.ID,
			Driver:     buptypes.VolumeDriverKindLocalFs,
			DriverOpts: "/go/src/github.com/function61/bup/__volume/1/",
		},
		&buptypes.VolumeMount{
			ID:         buputils.NewVolumeMountId(),
			Volume:     volume2.ID,
			Node:       newNode.ID,
			Driver:     buptypes.VolumeDriverKindLocalFs,
			DriverOpts: "/go/src/github.com/function61/bup/__volume/2/",
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
