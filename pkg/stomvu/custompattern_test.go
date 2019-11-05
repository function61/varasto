package stomvu

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestCustomMonthlyPattern(t *testing.T) {
	phoneCallPattern := customMonthlyPattern("^[0-9a-f]{2}([0-9]{14})p", "20060102150405")

	cs := []struct {
		input  string
		expect string
	}{
		{"0d20190620121528p+358504123456.m4a", "2019/06"},
		{"0d20181220121528p+358504123456.m4a", "2018/12"},
		{"0d20181320121528p+358504123456.m4a", ""}, // there is no 13th month => invalid
	}

	for _, c := range cs {
		t.Run(c.input, func(t *testing.T) {
			assert.EqualString(t, phoneCallPattern(c.input), c.expect)
		})
	}
}
