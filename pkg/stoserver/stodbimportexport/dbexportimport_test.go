package stodbimportexport

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestBackupHeaderWritingAndParsing(t *testing.T) {
	backupHeader := makeBackupHeader(backupHeaderJson{NodeId: "RH7j", SchemaVersion: 314})

	assert.EqualString(t, backupHeader, `# Varasto-DB-snapshot{"node_id":"RH7j","schema_version":314}`)

	details, err := parseBackupHeader(backupHeader)
	assert.Assert(t, err == nil)

	assert.EqualString(t, details.NodeId, "RH7j")
	assert.Assert(t, details.SchemaVersion == 314)
}
