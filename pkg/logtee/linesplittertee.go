package logtee

import (
	"io"
	"strings"
	"sync"
)

type lineSplitterTee struct {
	buf           []byte // buffer before receiving \n
	lineCompleted func(string)
	mu            sync.Mutex
}

// returns io.Writer that tees full lines to lineCompleted callback
func NewLineSplitterTee(sink io.Writer, lineCompleted func(string)) io.Writer {
	return io.MultiWriter(sink, &lineSplitterTee{
		buf:           []byte{},
		lineCompleted: lineCompleted,
	})
}

func (l *lineSplitterTee) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf = append(l.buf, data...)

	// as long as we have lines, chop the buffer down
	for {
		idx := strings.IndexByte(string(l.buf), '\n')
		if idx == -1 {
			break
		}

		l.lineCompleted(string(l.buf[0:idx]))

		l.buf = l.buf[idx+1:]
	}

	return len(data), nil
}
