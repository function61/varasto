package stodb

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

var (
	CfgNodeId              = configAccessor("nodeId")
	CfgTheMovieDbApikey    = configAccessor(stoservertypes.CfgTheMovieDbApikey)
	CfgIgdbApikey          = configAccessor(stoservertypes.CfgIgdbApikey)
	CfgFuseServerBaseUrl   = configAccessor(stoservertypes.CfgFuseServerBaseUrl)
	CfgNetworkShareBaseUrl = configAccessor(stoservertypes.CfgNetworkShareBaseUrl)
	CfgUbackupConfig       = configAccessor(stoservertypes.CfgUbackupConfig)
	CfgUpdateStatusAt      = configAccessor(stoservertypes.CfgUpdateStatusAt)
	CfgNodeTlsCertKey      = configAccessor(stoservertypes.CfgNodeTlsCertKey)
	CfgGrafanaUrl          = configAccessor(stoservertypes.CfgGrafanaUrl)
)
