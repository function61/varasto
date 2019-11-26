package stofuse

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestEncodeAndParseDirRef(t *testing.T) {
	combined := encodeDirRef("r-iZ5J_lXUI", "Dankest memes")

	assert.EqualString(t, combined, "Dankest memes - r-iZ5J_lXUI")

	assert.EqualString(t, parseDirRef(combined), "r-iZ5J_lXUI")

	assert.EqualString(t, parseDirRef("r-iZ5J_lXUI"), "r-iZ5J_lXUI")
}
