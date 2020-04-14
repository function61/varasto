package docreference

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

// tests that each for each member of DocRef (e.g. "docs/example.md") a file exists. that
// makes it possible for us to link to markdown view in GitHub with confidence that the URL
// will not 404 if we move files around later and forget to update the ref
func TestDocsExistForDocRefs(t *testing.T) {
	for _, member := range stoservertypes.DocRefMembers {
		member := member // pin
		t.Run(string(member), func(t *testing.T) {
			exists, err := fileexists.Exists("../../" + string(member))
			assert.Ok(t, err)
			assert.Assert(t, exists)
		})
	}
}

func TestGitHubMaster(t *testing.T) {
	assert.EqualString(
		t,
		GitHubMaster(stoservertypes.DocRefREADMEMd),
		"https://github.com/function61/varasto/blob/master/README.md")
}
