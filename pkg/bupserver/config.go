package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
)

type ServerConfig struct {
	SelfNode          buptypes.Node
	ClientsAuthTokens map[string]bool
}

func readConfigFromDatabase(db *storm.DB) (*ServerConfig, error) {
	var nodeId string
	if err := db.Get("settings", "nodeId", &nodeId); err != nil {
		return nil, err
	}

	var selfNode buptypes.Node
	if err := db.One("ID", nodeId, &selfNode); err != nil {
		return nil, err
	}

	authTokens := map[string]bool{}

	clients := []buptypes.Client{}
	if err := db.All(&clients); err != nil {
		return nil, err
	}

	for _, client := range clients {
		authTokens[client.AuthToken] = true
	}

	return &ServerConfig{
		SelfNode:          selfNode,
		ClientsAuthTokens: authTokens,
	}, nil
}
