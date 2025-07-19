package stodb

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

var (
	CfgNodeID              = configAccessor("nodeId")
	CfgTheMovieDBApikey    = configAccessor(stoservertypes.CfgTheMovieDbApikey)
	CfgIgdbAPIkey          = configAccessor(stoservertypes.CfgIgdbApikey)
	CfgFuseServerBaseURL   = configAccessor(stoservertypes.CfgFuseServerBaseUrl)
	CfgNetworkShareBaseURL = configAccessor(stoservertypes.CfgNetworkShareBaseUrl)
	CfgUbackupConfig       = configAccessor(stoservertypes.CfgUbackupConfig)
	CfgUpdateStatusAt      = configAccessor(stoservertypes.CfgUpdateStatusAt)
	CfgNodeTLSCertKey      = configAccessor(stoservertypes.CfgNodeTlsCertKey)
	CfgGrafanaURL          = configAccessor(stoservertypes.CfgGrafanaUrl)
	CfgMediascannerState   = configAccessor(stoservertypes.CfgMediascannerState)
)
