package logtee

import (
	"bytes"
	"fmt"
	"github.com/function61/gokit/assert"
	"testing"
)

func TestComposite(t *testing.T) {
	sink := &bytes.Buffer{}

	tail := NewStringTail(4)

	// writes to upstream all end up in the sink, but Snapshot() only returns the last 4 lines
	upstream := NewLineSplitterTee(sink, func(line string) {
		tail.Write(line)
	})

	_, _ = upstream.Write([]byte("line 1\nline 2\nline 3 left open"))

	assert.EqualString(t, fmt.Sprintf("%v", tail.Snapshot()), "[line 1 line 2]")

	_, _ = upstream.Write([]byte("\n")) // close line 3

	assert.EqualString(t, fmt.Sprintf("%v", tail.Snapshot()), "[line 1 line 2 line 3 left open]")

	_, _ = upstream.Write([]byte("line 4\nline 5\nline 6\n"))

	assert.EqualString(t, fmt.Sprintf("%v", tail.Snapshot()), "[line 3 left open line 4 line 5 line 6]")

	assert.EqualString(t, sink.String(), "line 1\nline 2\nline 3 left open\nline 4\nline 5\nline 6\n")
}
