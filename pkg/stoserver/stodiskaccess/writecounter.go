package stodiskaccess

import (
	"io"
)

// simply keeps track of how many bytes were written to it.
// one use case is to tee a io.Reader to this counter so a consumer of the reader does not
// have to report the count back to you, but you can capture it out-of-band
type writeCounter struct {
	count int64
}

func (c *writeCounter) BytesWritten() int64 {
	return c.count
}

func (c *writeCounter) Write(data []byte) (int, error) {
	l := len(data)
	c.count += int64(l)
	return l, nil
}

func (c *writeCounter) Tee(source io.Reader) io.Reader {
	return io.TeeReader(source, c)
}
