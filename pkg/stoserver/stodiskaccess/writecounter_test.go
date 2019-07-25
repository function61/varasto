package stodiskaccess

import (
	"github.com/function61/gokit/assert"
	"io/ioutil"
	"strings"
	"testing"
)

func TestWriteCounter(t *testing.T) {
	counter := &writeCounter{}

	msg, err := ioutil.ReadAll(counter.Tee(strings.NewReader("The quick brown fox jumps over the lazy dog")))
	assert.Assert(t, err == nil)

	assert.EqualString(t, string(msg), "The quick brown fox jumps over the lazy dog")
	assert.Assert(t, counter.BytesWritten() == 43)
}
