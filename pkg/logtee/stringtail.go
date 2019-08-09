package logtee

import (
	"container/ring"
	"sync"
)

type StringTail struct {
	lines *ring.Ring
	mu    sync.Mutex
}

// keeps only "capacity" last Write() calls (which you can retrieve with Snapshot() )
func NewStringTail(capacity int) *StringTail {
	r := ring.New(capacity)
	for i := 0; i < capacity; i++ { // init items
		r.Value = ""
		r = r.Next()
	}

	return &StringTail{
		lines: r,
	}
}

func (t *StringTail) Snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	ret := []string{}

	r := t.lines

	ll := r.Len()
	for i := 0; i < ll; i++ {
		val := r.Value.(string)
		if val != "" { // not sure about this check
			ret = append(ret, val)
		}
		r = r.Next()
	}

	return ret
}

func (t *StringTail) Write(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lines.Value = line
	t.lines = t.lines.Next()
}
