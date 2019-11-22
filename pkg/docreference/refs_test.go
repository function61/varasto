package docreference

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"testing"
)

// tests that each docs/example.md file exist for each member of DocRef. that makes it
// possible for us to link to markdown view in GitHub with confidence that the URL will
// not 404 if we move files around later and forget to update the ref
func TestDocsExistForDocRefs(t *testing.T) {
	for _, member := range stoservertypes.DocRefMembers {
		member := member // pin
		t.Run(string(member), func(t *testing.T) {
			exists, err := fileexists.Exists("../../" + string(member))
			if err != nil {
				panic(err)
			}

			assert.Assert(t, exists)
		})
	}
}

func TestGitHubMaster(t *testing.T) {
	assert.EqualString(
		t,
		GitHubMaster(stoservertypes.DocRefDocsGuideSettingUpBackupMd),
		"https://github.com/function61/varasto/blob/master/docs/guide_setting-up-backup.md")
}
