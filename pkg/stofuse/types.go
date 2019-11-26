package stofuse

import (
	"bazil.org/fuse/fs"
)

type DirContract interface {
	fs.Node
	fs.NodeStringLookuper
}

// we don't strictly need this, but it's easier to grasp what the interface is since
// the design of Bazil is so mix-n-match
type FileCustomInterface interface {
	fs.Node
	fs.HandleReader
}

type sigFabric struct {
	unmountAll chan interface{}
}

func newSigs() *sigFabric {
	return &sigFabric{
		unmountAll: make(chan interface{}),
	}
}
