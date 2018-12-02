package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/gokit/cryptorandombytes"
	"log"
)

func readConfigFromDatabaseOrBootstrapIfNeeded(db *storm.DB) (*ServerConfig, error) {
	// using this as a flag to check if boostrapping has been done before
	serverConfig, err := readConfigFromDatabase(db)
	if err == nil {
		return serverConfig, nil
	}

	// we have error => possibly need bootstrap

	// totally unexpected error?
	if err != storm.ErrNotFound {
		return nil, err
	}

	// was not found error => run bootstrap
	if err := bootstrap(db); err != nil {
		return nil, err
	}

	return readConfigFromDatabase(db)
}

func bootstrap(db *storm.DB) error {
	nodeId := buputils.NewNodeId()
	authToken := cryptorandombytes.Base64Url(32)

	log.Printf("generated nodeId: %s", nodeId)

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	volume1 := buptypes.Volume{
		ID:         buputils.NewVolumeId(),
		Label:      "dev-volume",
		Driver:     buptypes.VolumeDriverKindLocalFs,
		DriverOpts: "/go/src/github.com/function61/bup/__volume/1/",
	}

	volume2 := buptypes.Volume{
		ID:         buputils.NewVolumeId(),
		Label:      "dev-volume",
		Driver:     buptypes.VolumeDriverKindLocalFs,
		DriverOpts: "/go/src/github.com/function61/bup/__volume/2/",
	}

	replPolicy := buptypes.ReplicationPolicy{
		ID:             "default",
		Name:           "Default replication policy",
		DesiredVolumes: []string{volume1.ID, volume2.ID},
	}

	newNode := buptypes.Node{
		ID:              nodeId,
		Addr:            "localhost:8066",
		Name:            "dev",
		AccessToVolumes: []string{volume1.ID, volume2.ID},
	}

	if err := tx.Save(&newNode); err != nil {
		return err
	}

	if err := tx.Save(&volume1); err != nil {
		return err
	}

	if err := tx.Save(&volume2); err != nil {
		return err
	}

	if err := tx.Save(&replPolicy); err != nil {
		return err
	}

	if err := tx.Set("settings", "nodeId", &nodeId); err != nil {
		return err
	}

	if err := tx.Set("settings", "authToken", &authToken); err != nil {
		return err
	}

	return tx.Commit()
}
