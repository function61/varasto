package stofuse

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestMkFsSafe(t *testing.T) {
	assert.EqualString(t, mkFsSafe("Police Academy: Mission to Moscow"), "Police Academy_ Mission to Moscow")

	assert.EqualString(
		t,
		mkFsSafe(`All special chars = \ and / and : and * and ? and " and < and > and |`),
		"All special chars = _ and _ and _ and _ and _ and _ and _ and _ and _")
}
