package stodb

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

var (
	CfgNodeId            = ConfigAccessor("nodeId")
	CfgTheMovieDbApikey  = ConfigAccessor(stoservertypes.CfgTheMovieDbApikey)
	CfgFuseServerBaseUrl = ConfigAccessor(stoservertypes.CfgFuseServerBaseUrl)
)
