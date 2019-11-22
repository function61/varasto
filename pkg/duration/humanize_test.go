package duration

import (
	"github.com/function61/gokit/assert"
	"testing"
	"time"
)

func TestHumanize(t *testing.T) {
	tcs := []struct {
		input  string
		output string
	}{
		{"0ms", "0 milliseconds"},
		{"1ms", "1 millisecond"},
		{"499ms", "499 milliseconds"},
		{"500ms", "1 second"},
		{"1s", "1 second"},
		{"29s", "29 seconds"},
		{"30s", "1 minute"},
		{"29m", "29 minutes"},
		{"36m34.20996749s", "1 hour"},
		{"89m", "1 hour"},
		{"90m", "2 hours"},
		{"12h", "1 day"},
		{"36h", "2 days"},
	}

	for _, tc := range tcs {
		tc := tc // pin

		t.Run(tc.input, func(t *testing.T) {
			dur, err := time.ParseDuration(tc.input)
			assert.Assert(t, err == nil)

			assert.EqualString(t, Humanize(dur), tc.output)
		})
	}
}
