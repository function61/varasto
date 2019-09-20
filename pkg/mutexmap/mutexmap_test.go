package mutexmap

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestMutexMap(t *testing.T) {
	mm := New()

	releaseFoo, fooOk := mm.TryLock("foo")
	assert.Assert(t, fooOk)

	_, fooConcurrentOk := mm.TryLock("foo")
	assert.Assert(t, !fooConcurrentOk)

	releaseFoo()

	releaseFoo, fooOk = mm.TryLock("foo")
	assert.Assert(t, fooOk)
	defer releaseFoo()
}
