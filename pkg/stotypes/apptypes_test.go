package stotypes

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestEqual(t *testing.T) {
	a, _ := BlobRefFromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	b, _ := BlobRefFromHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	assert.Assert(t, a.Equal(*a))
	assert.Assert(t, !a.Equal(*b))
}
