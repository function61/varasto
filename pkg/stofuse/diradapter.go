package stofuse

// adapts a directory in Varasto to FUSE directory.
// a Varasto directory can have subdirectories and collections.
// both are projected as symlinks to further Varasto-FUSE "queries".
//
// TODO: implement "$ mkdir"?

import (
	"context"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

type dirAdapter struct {
	dirID   string
	srv     *FsServer
	inode   uint64
	cacheMu sync.Mutex

	entriesCached []stoEntry
}

var _ interface {
	fs.Node
	fs.NodeStringLookuper
} = (*dirAdapter)(nil)

func NewDirAdapter(dirID string, srv *FsServer) *dirAdapter {
	return &dirAdapter{
		dirID: dirID,
		srv:   srv,
		inode: nextInode(),

		entriesCached: nil, // purposefully nil so we can detect uncached situation
	}
}

func (b *dirAdapter) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = b.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (b *dirAdapter) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if _, err := b.ReadDirAll(ctx); err != nil { // calling just for the side effect to fill cache
		return nil, err
	}

	for _, entry := range b.entriesCached {
		if entry.dirent.Name == name {
			return entry.node, nil
		}
	}

	return nil, fuse.ENOENT
}

func (b *dirAdapter) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// FIXME: does this lock work properly?
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	if err := b.computeCache(ctx); err != nil {
		b.srv.logl.Error.Printf("dirAdapter.ReadDirAll: %v", err)
		return nil, fuse.ENOENT
	}

	dents := []fuse.Dirent{}
	for _, entry := range b.entriesCached {
		dents = append(dents, entry.dirent)
	}

	return dents, nil
}

// caller has acquired mutex
func (b *dirAdapter) computeCache(ctx context.Context) error {
	if b.entriesCached != nil { // FIXME: bad
		return nil
	}

	conf := b.srv.client.Config()

	dirOutput := &stoservertypes.DirectoryOutput{}
	if _, err := ezhttp.Get(
		ctx,
		conf.URLBuilder().GetDirectory(b.dirID),
		ezhttp.AuthBearer(conf.AuthToken),
		ezhttp.RespondsJson(dirOutput, false),
		ezhttp.Client(conf.HTTPClient()),
	); err != nil {
		return err
	}

	b.entriesCached = []stoEntry{}

	for _, collectionWithMeta := range dirOutput.Collections {
		coll := collectionWithMeta.Collection // shorthand

		b.entriesCached = append(b.entriesCached, newStoSymlink(b.srv.AbsoluteSymlinkUnderFUSEMount("id", coll.Id)).MakeStoEntry(coll.Name))
	}

	for _, dir := range dirOutput.SubDirectories {
		b.entriesCached = append(b.entriesCached, newStoSymlink(b.srv.AbsoluteSymlinkUnderFUSEMount("dir", dir.Directory.Id)).MakeStoEntry(dir.Directory.Name))
	}

	return nil
}
