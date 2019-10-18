package docreference

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func GitHubMaster(ref stoservertypes.DocRef) string {
	return "https://github.com/function61/varasto/blob/master/" + string(ref)
}
