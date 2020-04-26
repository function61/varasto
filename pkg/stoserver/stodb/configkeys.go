package stodb

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

var (
	CfgNodeId              = ConfigAccessor("nodeId")
	CfgTheMovieDbApikey    = ConfigAccessor(stoservertypes.CfgTheMovieDbApikey)
	CfgIgdbApikey          = ConfigAccessor(stoservertypes.CfgIgdbApikey)
	CfgFuseServerBaseUrl   = ConfigAccessor(stoservertypes.CfgFuseServerBaseUrl)
	CfgNetworkShareBaseUrl = ConfigAccessor(stoservertypes.CfgNetworkShareBaseUrl)
	CfgUbackupConfig       = ConfigAccessor(stoservertypes.CfgUbackupConfig)
	CfgUpdateStatusAt      = ConfigAccessor(stoservertypes.CfgUpdateStatusAt)
	CfgNodeTlsCertKey      = ConfigAccessor(stoservertypes.CfgNodeTlsCertKey)
)
