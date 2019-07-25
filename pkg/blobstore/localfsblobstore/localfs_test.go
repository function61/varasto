package localfsblobstore

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"testing"
)

func TestPath(t *testing.T) {
	driver := New("APvMjudT4IQ", "/tmp/", nil)

	blobRef, _ := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")

	assert.EqualString(t,
		driver.getPath(*blobRef),
		"/tmp/d7/a/8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592.chunk")
}
