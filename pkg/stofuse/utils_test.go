package stofuse

import (
	"fmt"
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"strings"
	"testing"
)

func TestAlignReads(t *testing.T) {
	serialize := func(brs []alignedBlobRead) string {
		lines := []string{}

		for _, br := range brs {
			lines = append(lines, fmt.Sprintf(
				"blob<%d> offset<%d> len<%d>",
				br.blobIdx,
				br.offsetInBlob,
				br.lenInBlob))
		}

		return strings.Join(lines, "\n")
	}

	assert.EqualString(t, serialize(alignReads(0, 100)), "blob<0> offset<0> len<100>")
	assert.EqualString(t, serialize(alignReads(stotypes.BlobSize-1, 2)), "blob<0> offset<4194303> len<1>\nblob<1> offset<0> len<1>")
	assert.EqualString(t, serialize(alignReads(stotypes.BlobSize-1, stotypes.BlobSize+1)), "blob<0> offset<4194303> len<1>\nblob<1> offset<0> len<4194304>")
	assert.EqualString(t, serialize(alignReads(stotypes.BlobSize-1, stotypes.BlobSize+2)), "blob<0> offset<4194303> len<1>\nblob<1> offset<0> len<4194304>\nblob<2> offset<0> len<1>")
}

func TestMkFsSafe(t *testing.T) {
	assert.EqualString(t, mkFsSafe("Police Academy: Mission to Moscow"), "Police Academy_ Mission to Moscow")

	assert.EqualString(
		t,
		mkFsSafe(`All special chars = \ and / and : and * and ? and " and < and > and |`),
		"All special chars = _ and _ and _ and _ and _ and _ and _ and _ and _")
}
