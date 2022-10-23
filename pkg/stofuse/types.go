package stofuse

import (
	"context"
	stdfs "io/fs"

	"bazil.org/fuse"
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

// couples together a directory entry ("node name") and a node
type stoEntry struct {
	dirent fuse.Dirent // name found from here
	node   fs.Node
}

func newStoEntry(dirent fuse.Dirent, node fs.Node) stoEntry {
	return stoEntry{dirent, node}
}

type stoSymlink struct {
	target string
	inode  uint64
}

func (s *stoSymlink) MakeDirent(name string) fuse.Dirent {
	return fuse.Dirent{
		Inode: s.inode,
		Name:  name,
		Type:  fuse.DT_Link,
	}
}

func (s *stoSymlink) MakeStoEntry(name string) stoEntry {
	return stoEntry{s.MakeDirent(name), s}
}

func newStoSymlink(target string) *stoSymlink {
	return &stoSymlink{
		target: target,
		inode:  nextInode(),
	}
}

func (s *stoSymlink) Readlink(_ context.Context, req *fuse.ReadlinkRequest) (string, error) {
	return s.target, nil
}

func (s *stoSymlink) Attr(_ context.Context, attr *fuse.Attr) error {
	attr.Inode = s.inode
	attr.Mode = stdfs.ModeSymlink | 0555

	return nil
}

var _ interface {
	fs.NodeReadlinker
	fs.Node
} = (*stoSymlink)(nil)
