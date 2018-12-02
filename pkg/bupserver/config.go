package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
)

type ServerConfig struct {
	SelfNode         buptypes.Node
	ClientsAuthToken string
}

func readConfigFromDatabase(db *storm.DB) (*ServerConfig, error) {
	var nodeId string
	if err := db.Get("settings", "nodeId", &nodeId); err != nil {
		return nil, err
	}

	var authToken string
	if err := db.Get("settings", "authToken", &authToken); err != nil {
		return nil, err
	}

	var selfNode buptypes.Node
	if err := db.One("ID", nodeId, &selfNode); err != nil {
		return nil, err
	}

	return &ServerConfig{
		SelfNode:         selfNode,
		ClientsAuthToken: authToken,
	}, nil
}
