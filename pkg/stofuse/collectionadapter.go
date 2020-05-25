package stofuse

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stotypes"
)

func adaptCollectionToDirectory(coll *stotypes.Collection, srv *FsServer) (*CollectionAsDir, error) {
	state, err := stateresolver.ComputeStateAtHead(*coll)
	if err != nil {
		return nil, err
	}

	collectionRoot := adaptCollectionToDirectoryInternal(state.FileList(), ".", coll.ID, srv)
	collectionRoot.collection = coll

	// collection's root dir has always dot as name, fix it
	collectionRoot.name = mkFsSafe(coll.Name)

	return collectionRoot, nil
}

func adaptCollectionToDirectoryInternal(
	dirFiles []stotypes.File,
	pathForDirpeek string,
	collectionId string,
	srv *FsServer,
) *CollectionAsDir {
	dpr := stateresolver.DirPeek(dirFiles, pathForDirpeek)

	rootFiles := []*CollectionAsDirFile{}

	for _, file := range dpr.Files {
		f := NewCollectionAsDirFile(
			mkFsSafe(filepath.Base(file.Path)),
			nextInode(),
			file,
			collectionId,
			srv)

		rootFiles = append(rootFiles, f)
	}

	subDirs := []*CollectionAsDir{}

	for _, subDirPath := range dpr.SubDirs {
		subDir := adaptCollectionToDirectoryInternal(dirFiles, subDirPath, collectionId, srv)

		subDirs = append(subDirs, subDir)
	}

	return NewCollectionAsDir(filepath.Base(pathForDirpeek), nextInode(), rootFiles, subDirs)
}

func NewCollectionAsDir(name string, inode uint64, files []*CollectionAsDirFile, subdirs []*CollectionAsDir) *CollectionAsDir {
	return &CollectionAsDir{
		name:    name,
		inode:   inode,
		files:   files,
		subdirs: subdirs,
	}
}

type CollectionAsDir struct {
	name       string
	collection *stotypes.Collection
	inode      uint64
	files      []*CollectionAsDirFile
	subdirs    []*CollectionAsDir
}

func (d CollectionAsDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d CollectionAsDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, dir := range d.subdirs {
		if dir.name == name {
			return dir, nil
		}
	}

	for _, file := range d.files {
		if file.name == name {
			return file, nil
		}
	}

	return nil, fuse.ENOENT
}

func (d CollectionAsDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := []fuse.Dirent{}

	for _, subdir := range d.subdirs {
		entries = append(entries, fuse.Dirent{
			Inode: subdir.inode,
			Name:  subdir.name,
			Type:  fuse.DT_Dir,
		})
	}

	for _, file := range d.files {
		entries = append(entries, fuse.Dirent{
			Inode: file.inode,
			Name:  file.name,
			Type:  fuse.DT_File,
		})
	}

	return entries, nil
}

func NewCollectionAsDirFile(
	name string,
	inode uint64,
	file stotypes.File,
	collectionId string,
	srv *FsServer,
) *CollectionAsDirFile {
	return &CollectionAsDirFile{inode, file, name, collectionId, srv}
}

var _ FileCustomInterface = (*CollectionAsDirFile)(nil)

type CollectionAsDirFile struct {
	inode        uint64
	file         stotypes.File
	name         string
	collectionId string
	srv          *FsServer
}

func (f CollectionAsDirFile) Attr(ctx context.Context, attrs *fuse.Attr) error {
	attrs.Inode = f.inode
	attrs.Mode = 0444
	attrs.Mtime = f.file.Modified
	attrs.Size = uint64(f.file.Size)

	return nil
}

func (f CollectionAsDirFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// nothing to offer, already past file limit
	if req.Offset >= f.file.Size {
		return errors.New("offset past EOF")
	}

	for _, alignedRead := range alignReads(req.Offset, min(int64(req.Size), f.file.Size-req.Offset)) {
		blobRef, err := stotypes.BlobRefFromHex(f.file.BlobRefs[alignedRead.blobIdx])
		if err != nil {
			return err
		}

		bd, err := f.srv.blobCache.Get(ctx, *blobRef, f.collectionId)
		if err != nil {
			return err
		}

		resp.Data = append(resp.Data, bd.Data[alignedRead.offsetInBlob:int64(alignedRead.offsetInBlob)+alignedRead.lenInBlob]...)
	}

	return nil
}
