// +build !windows

package stofuse

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"errors"
	"github.com/function61/gokit/retry"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type sigFabric struct {
	mount      chan string
	unmount    chan string
	unmountAll chan interface{}
}

func newSigs() *sigFabric {
	return &sigFabric{
		mount:      make(chan string),
		unmount:    make(chan string),
		unmountAll: make(chan interface{}),
	}
}

func fuseServe(sigs *sigFabric, conf stoclient.ClientConfig, stop *stopper.Stopper) error {
	defer stop.Done()

	if conf.FuseMountPath == "" {
		return errors.New("FuseMountPath not set")
	}

	NewFsServer(conf)

	varastoFs, err := NewVarastoFS(sigs)
	if err != nil {
		return err
	}

	// AllowOther() needed to get our Samba use case working
	fuseConn, err := fuse.Mount(
		conf.FuseMountPath,
		// fuse.ReadOnly(),
		fuse.FSName("varasto"),
		fuse.Subtype("varasto1fs"),
		fuse.AllowOther())
	if err != nil {
		return err
	}
	defer fuseConn.Close()

	go func() {
		mountAdd := func(collectionId string) error {
			coll, err := stoclient.FetchCollectionMetadata(globalFsServer.clientConfig, collectionId)
			if err != nil {
				return err
			}

			state, err := stateresolver.ComputeStateAt(*coll, coll.Head)
			if err != nil {
				return err
			}

			collectionRoot := processOneDir(state.FileList(), ".")
			collectionRoot.collection = coll

			// collection's root dir has always dot as name, fix it
			collectionRoot.name = mkFsSafe(coll.Name)

			root := varastoFs.root
			root.subdirs = append(root.subdirs, collectionRoot)

			return nil
		}

		mountRemove := func(collectionId string) error {
			subdirsWithCollectionRemoved := []*Dir{}
			for _, dir := range varastoFs.root.subdirs {
				if dir.collection.ID != collectionId {
					subdirsWithCollectionRemoved = append(subdirsWithCollectionRemoved, dir)
				}
			}

			varastoFs.root.subdirs = subdirsWithCollectionRemoved

			return nil
		}

		for {
			select {
			case <-stop.Signal:
				break
			case collectionId := <-sigs.mount:
				if err := mountAdd(collectionId); err != nil {
					log.Printf("ERROR: mountAdd: %v", err)
				} else {
					log.Printf("mount: %s", collectionId)
				}
			case collectionId := <-sigs.unmount:
				if err := mountRemove(collectionId); err != nil {
					log.Printf("ERROR: mountRemove: %v", err)
				} else {
					log.Printf("unmount: %s", collectionId)
				}
			case <-sigs.unmountAll:
				varastoFs.root.subdirs = []*Dir{}
			}
		}
	}()

	go func() {
		<-stop.Signal

		tryUnmount := func(ctx context.Context) error {
			// "Instead of sending a SIGINT to the process, you should unmount the filesystem
			// That will cause the serve loop to exit, and your process can exit that way."
			// https://github.com/bazil/fuse/issues/6
			return fuse.Unmount(conf.FuseMountPath)
		}

		ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
		defer cancel()

		// retrying because unmount will fail if any process is accessing the mount
		if err := retry.Retry(ctx, tryUnmount, retry.DefaultBackoff(), func(err error) {
			log.Printf("tryUnmount: %v", err)
		}); err != nil {
			panic(err)
		}
	}()

	log.Printf("VarastoFS server started")

	if err := fs.Serve(fuseConn, varastoFs); err != nil {
		return err
	}

	// check if the mount process has an error to report
	// blocks as long as server is up
	<-fuseConn.Ready
	if err := fuseConn.MountError; err != nil {
		log.Fatal(err)
	}

	return nil
}

type VarastoFSRoot struct {
	root *Dir
}

func processOneDir(dirFiles []stotypes.File, pathForDirpeek string) *Dir {
	dpr := stateresolver.DirPeek(dirFiles, pathForDirpeek)

	rootFiles := []*File{}

	for _, file := range dpr.Files {
		f := NewFile(
			mkFsSafe(filepath.Base(file.Path)),
			nextInode(),
			file)

		rootFiles = append(rootFiles, f)
	}

	subDirs := []*Dir{}

	for _, subDirPath := range dpr.SubDirs {
		subDir := processOneDir(dirFiles, subDirPath)

		subDirs = append(subDirs, subDir)
	}

	return NewDir(filepath.Base(pathForDirpeek), nextInode(), rootFiles, subDirs)
}

func NewVarastoFS(sigs *sigFabric) (*VarastoFSRoot, error) {
	root := NewDir("/", nextInode(), nil, []*Dir{})

	return &VarastoFSRoot{root}, nil
}

// implements fs.FS
func (f VarastoFSRoot) Root() (fs.Node, error) {
	return f.root, nil
}

/*
type DirContract interface {
	fs.Node
	fs.NodeStringLookuper
}
*/

func NewDir(name string, inode uint64, files []*File, subdirs []*Dir) *Dir {
	return &Dir{
		name:    name,
		inode:   inode,
		files:   files,
		subdirs: subdirs,
	}
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	name       string
	collection *stotypes.Collection
	inode      uint64
	files      []*File
	subdirs    []*Dir
}

func (d Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
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

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
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

// we don't strictly need this, but it's easier to grasp what the interface is since
// the design of Bazil is so mix-n-match
/*
type FileCustomInterface interface {
	fs.Node
	fs.HandleReader
}
*/

func NewFile(name string, inode uint64, file stotypes.File) *File {
	return &File{inode, file, name}
}

// File implements both Node and Handle for the hello file.
type File struct {
	inode uint64
	file  stotypes.File
	name  string
}

func (f File) Attr(ctx context.Context, attrs *fuse.Attr) error {
	attrs.Inode = f.inode
	attrs.Mode = 0444
	attrs.Mtime = f.file.Modified
	attrs.Size = uint64(f.file.Size)

	return nil
}

func (f File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// nothing to offer
	if req.Offset >= f.file.Size {
		return nil
	}

	chunkIdx, correctedOffset := stoclient.BlobIdxFromOffset(uint64(req.Offset))

	blobRef, err := stotypes.BlobRefFromHex(f.file.BlobRefs[chunkIdx])
	if err != nil {
		return err
	}

	bd, err := globalFsServer.blobCache.Get(ctx, *blobRef)
	if err != nil {
		return err
	}

	end := correctedOffset + int64(req.Size)

	if end > f.file.Size {
		end = f.file.Size
	}

	resp.Data = bd.Data[correctedOffset:end]
	return nil
}

// https://serverfault.com/a/650041
// \ / : * ? " < > |
var fsWindowsUnsafeRe = regexp.MustCompile("[\\\\/:*?\"<>|]")

func mkFsSafe(input string) string {
	return fsWindowsUnsafeRe.ReplaceAllString(input, "_")
}
