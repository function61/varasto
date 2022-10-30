package stofuse

// /sto/dir/<id> => queries directory dynamically from Varasto and projects it as directory adapter
// which itself projects subdirs and collections as symlinks to further directory or collection queries.

import (
	"context"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/varasto/pkg/mutexmap"
)

type dirByIDQuery struct {
	srv          *FsServer
	inode        uint64
	fetchByDirID *mutexmap.M
	cache        map[string]fs.Node
	cacheDents   []fuse.Dirent
	cacheMu      sync.Mutex
}

var _ interface {
	fs.Node
	fs.NodeStringLookuper
} = (*dirByIDQuery)(nil)

func NewDirByIDQuery(srv *FsServer) *dirByIDQuery {
	return &dirByIDQuery{
		srv:          srv,
		inode:        nextInode(),
		fetchByDirID: mutexmap.New(),
		cache:        map[string]fs.Node{},
		cacheDents:   []fuse.Dirent{},
	}
}

func (b *dirByIDQuery) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = b.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (b *dirByIDQuery) Lookup(ctx context.Context, name string) (fs.Node, error) {
	dirID := name

	unlock := b.fetchByDirID.Lock(dirID)
	defer unlock()

	// cache check needs to be inside lock to prevent unnecessary double fetch
	node := b.getCached(dirID)
	if node != nil {
		return node, nil
	}

	dir := NewDirAdapter(dirID, b.srv)
	if _, err := dir.ReadDirAll(ctx); err != nil { // FIXME: not sure if all errors are ENOENT
		return nil, fuse.ENOENT
	}

	b.setCached(dirID, dir)

	return dir, nil
}

func (b *dirByIDQuery) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// FIXME: does this lock work properly?
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	return b.cacheDents, nil
}

func (b *dirByIDQuery) getCached(dirID string) fs.Node {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	return b.cache[dirID]
}

func (b *dirByIDQuery) setCached(dirID string, dir *dirAdapter) {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	b.cache[dirID] = dir
	b.cacheDents = append(b.cacheDents, fuse.Dirent{
		Inode: dir.inode,
		Name:  dirID,
		Type:  fuse.DT_Dir,
	})
}
