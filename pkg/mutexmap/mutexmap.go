package mutexmap

import (
	"sync"
)

// Think of this as an infinite number of named bathroom stalls. Each named stall can only
// be occupied by one person.
// When you TryLock():
// a) it won't open if it's already occupied. (because it's locked inside)
// b) it opens and you get in and the stall gets reserved/locked for you. When you get out
//    you call the unlock callback you obtained from TryLock() to return the stall for use.
type M struct {
	// we need mutex around a map anyway, and our current requirements only need TryLock(),
	// so we can space-efficiently implement these with just boolean lock statuses..
	locks    map[string]bool
	masterMu sync.Mutex
}

func New() *M {
	return &M{
		locks: map[string]bool{},
	}
}

// returns false if gate already open/reserved
// returns true if gate was opened for you. you have to use the returned func to release it
func (n *M) TryLock(key string) (func(), bool) {
	n.masterMu.Lock()
	defer n.masterMu.Unlock()

	if _, open := n.locks[key]; open {
		return func() {}, false
	}

	n.locks[key] = true

	return func() {
		n.masterMu.Lock()
		defer n.masterMu.Unlock()

		delete(n.locks, key)
	}, true
}
