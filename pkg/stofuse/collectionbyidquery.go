package stofuse

import (
	"context"
	"os"
	"regexp"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/varasto/pkg/mutexmap"
)

type byIdDir struct {
	srv           *FsServer
	inode         uint64
	fetchByCollId *mutexmap.M
	cache         map[string]fs.Node
	cacheDents    []fuse.Dirent
	cacheMu       sync.Mutex
}

var _ DirContract = (*byIdDir)(nil)

func NewByIdDir(srv *FsServer) *byIdDir {
	return &byIdDir{
		srv:           srv,
		inode:         nextInode(),
		fetchByCollId: mutexmap.New(),
		cache:         map[string]fs.Node{},
		cacheDents:    []fuse.Dirent{},
	}
}

// internally means emptying caches, so previously used collections
// won't show up as directory listing
func (b *byIdDir) ForgetDirs() {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	b.cache = map[string]fs.Node{}
	b.cacheDents = []fuse.Dirent{}
}

func (b *byIdDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = b.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (b *byIdDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	collId := parseDirRef(name)

	unlock := b.fetchByCollId.Lock(collId)
	defer unlock()

	// cache check needs to be inside lock to prevent unnecessary double fetch
	node := b.getCached(collId)
	if node != nil {
		return node, nil
	}

	collection, err := b.srv.client.FetchCollectionMetadata(ctx, collId)
	if err != nil {
		return nil, fuse.ENOENT
	}

	dir, err := adaptCollectionToDirectory(collection, b.srv)
	if err != nil {
		return nil, err
	}

	b.setCached(collId, dir)

	return dir, nil
}

func (b *byIdDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// FIXME: does this lock work properly?
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	return b.cacheDents, nil
}

func (b *byIdDir) getCached(collId string) fs.Node {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	return b.cache[collId]
}

func (b *byIdDir) setCached(collId string, cad *CollectionAsDir) {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	b.cache[collId] = cad
	b.cacheDents = append(b.cacheDents, fuse.Dirent{
		Inode: cad.inode,
		Name:  encodeDirRef(collId, cad.name),
		Type:  fuse.DT_Dir,
	})
}

func encodeDirRef(id string, name string) string {
	return name + " - " + id
}

var parseDirRefRe = regexp.MustCompile(` - ([a-zA-Z0-9_\-]+)$`)

func parseDirRef(dirRef string) string {
	match := parseDirRefRe.FindStringSubmatch(dirRef)
	if match == nil {
		return dirRef
	} else {
		return match[1]
	}
}
