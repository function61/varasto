package mutexmap

import (
	"testing"
	"time"

	"github.com/function61/gokit/assert"
)

func TestMutexMap(t *testing.T) {
	mm := New()

	unlockFoo := mm.Lock("foo")
	unlockFoo()

	unlockFoo, fooOk := mm.TryLock("foo")
	assert.Assert(t, fooOk)

	_, fooConcurrentOk := mm.TryLock("foo")
	assert.Assert(t, !fooConcurrentOk)

	unlockFoo()

	unlockFoo, fooOk = mm.TryLock("foo")
	assert.Assert(t, fooOk)

	lockAcquireDuration := make(chan time.Duration)

	go func() {
		startedAcquiringLock := time.Now()

		unlock := mm.Lock("foo")
		defer unlock()

		lockAcquireDuration <- time.Since(startedAcquiringLock)
	}()

	time.Sleep(11 * time.Millisecond)

	unlockFoo()

	assert.Assert(t, <-lockAcquireDuration > 10*time.Millisecond)
}
