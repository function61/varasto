package byteshuman

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestHumanize(t *testing.T) {
	for _, tc := range []struct {
		input  uint64
		output string
	}{
		{0, "0 B"},
		{1024, "1.00 kiB"},
		{1536.0, "1.50 kiB"},
		{1048576, "1.00 MiB"},
		{1572864, "1.50 MiB"},
		{1073741824, "1.00 GiB"},
		{1610612736, "1.50 GiB"},
		{1099511627776, "1.00 TiB"},
		{1649267441664, "1.50 TiB"},
		{1125899906842624, "1.00 PiB"},
		{1688849860263936, "1.50 PiB"},
		{1152921504606846976, "1024.00 PiB"},
	} {
		t.Run(tc.output, func(t *testing.T) {
			assert.EqualString(t, Humanize(tc.input), tc.output)
		})
	}
}
