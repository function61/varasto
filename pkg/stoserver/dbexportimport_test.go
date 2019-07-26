package stoserver

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestBackupHeaderWritingAndParsing(t *testing.T) {
	backupHeader := makeBackupHeader("RH7j")

	assert.EqualString(t, backupHeader, "# Varasto-backup-v1(nodeId=RH7j)")

	nodeId, err := parseBackupHeader(backupHeader)

	assert.Assert(t, err == nil)
	assert.EqualString(t, nodeId, "RH7j")
}
