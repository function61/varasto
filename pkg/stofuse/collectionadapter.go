package stofuse

// TODO: in edit cases we're mixing and mathing inodes from our in-process generator and the FS that
//       backs the workdir. will there be collisions?

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/pkg/xattr"
)

func adaptCollectionToDirectory(coll *stotypes.Collection, srv *FsServer) (*CollectionDirNode, error) {
	state, err := stateresolver.ComputeStateAtHead(*coll)
	if err != nil {
		return nil, err
	}

	collectionRoot := adaptCollectionToDirectoryInternal(coll, state.FileList(), ".", srv, newOverlayFiles())

	collectionRoot.varastoClientStateHEAD = createHeadState(coll)

	return collectionRoot, nil
}

// *fullPathFromRoot* = "." | "Movies" | "Movies/Titanic" | ...
func adaptCollectionToDirectoryInternal(
	collection *stotypes.Collection,
	dirFiles []stotypes.File,
	fullPathFromRoot string,
	srv *FsServer,
	overlay *overlayFiles,
) *CollectionDirNode {
	dpr := stateresolver.DirPeek(dirFiles, fullPathFromRoot)

	rootFiles := []*CollectionDirNodeFile{}

	for _, file := range dpr.Files {
		f := NewCollectionDirNodeFile(
			mkFsSafe(filepath.Base(file.Path)),
			nextInode(),
			file,
			collection.ID,
			srv)

		rootFiles = append(rootFiles, f)
	}

	subDirs := []*CollectionDirNode{}

	for _, subDirPath := range dpr.SubDirs { // *subDirPath* is full path from root, e.g. "Movies/Titanic"
		subDir := adaptCollectionToDirectoryInternal(collection, dirFiles, subDirPath, srv, overlay)

		subDirs = append(subDirs, subDir)
	}

	return newCollectionDirNode(collection, fullPathFromRoot, nextInode(), rootFiles, subDirs, overlay)
}

func NewCollectionDirNode(
	collection *stotypes.Collection,
	fullPathFromRoot string,
	inode uint64,
	filesCommitted []*CollectionDirNodeFile,
	subdirsCommitted []*CollectionDirNode,
) *CollectionDirNode {
	return newCollectionDirNode(collection, fullPathFromRoot, inode, filesCommitted, subdirsCommitted, newOverlayFiles())
}

func newCollectionDirNode(
	collection *stotypes.Collection,
	fullPathFromRoot string,
	inode uint64,
	filesCommitted []*CollectionDirNodeFile,
	subdirsCommitted []*CollectionDirNode,
	overlay *overlayFiles,
) *CollectionDirNode {
	return &CollectionDirNode{
		collection:       collection,
		dirBaseName:      filepath.Base(fullPathFromRoot),
		fullPathFromRoot: fullPathFromRoot,
		inode:            inode,
		filesCommitted:   filesCommitted,
		subdirsCommitted: subdirsCommitted,
		overlay:          overlay,
	}
}

type CollectionDirNode struct {
	mu                     sync.RWMutex
	dirBaseName            string      // "/" => ".", "/Movies/Titanic" => "Titanic"
	fullPathFromRoot       string      // "/" => ".", "/Movies/Titanic" => "Movies/Titanic"
	varastoClientStateHEAD *staticFile // only set if root
	collection             *stotypes.Collection
	inode                  uint64
	filesCommitted         []*CollectionDirNodeFile
	subdirsCommitted       []*CollectionDirNode
	overlay                *overlayFiles
}

var _ interface {
	fs.Node
	fs.NodeStringLookuper
	fs.NodeMkdirer
	fs.HandleReadDirAller
	fs.NodeCreater
	fs.NodeRemover
	fs.NodeRenamer
} = (*CollectionDirNode)(nil)

func (d *CollectionDirNode) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = d.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *CollectionDirNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if strings.Contains(name, "/") { // validate assumption that dirs are accessed one level at a time before reaching to subdir's file
		panic("Lookup: assumption failed")
	}

	if d.varastoClientStateHEAD != nil && name == stoclient.LocalStatefile {
		return d.varastoClientStateHEAD, nil
	}

	if stat, err := os.Stat(d.workdirPath(name)); err == nil { // look from uncommitted files
		if stat.IsDir() {
			return d.overlay.getOrCreateDir(d.workdirPath(name), func() *CollectionDirNode {
				return newCollectionDirNode(
					d.collection,
					filepath.Join(d.pathFromRoot(), name),
					stat.Sys().(*syscall.Stat_t).Ino,
					[]*CollectionDirNodeFile{},
					[]*CollectionDirNode{},
					d.overlay,
				)
			}), nil
		} else {
			return d.overlay.getOrCreate(d.workdirPath(name)), nil
		}
	}

	for _, dir := range d.subdirsCommitted {
		if dir.dirBaseName == name {
			return dir, nil
		}
	}

	for _, file := range d.filesCommitted {
		if file.name == name {
			return file, nil
		}
	}

	return nil, fuse.ENOENT
}

func (d *CollectionDirNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := []fuse.Dirent{}
	overlayEntries := []fuse.Dirent{}
	overlayNames := map[string]struct{}{}

	uncommitted, err := os.ReadDir(d.workdirPath(""))
	if err != nil {
		if os.IsNotExist(err) {
			// ignore (it's ok to have committed subdirs to which we don't have work subdir for)
		} else {
			return nil, fuse.EIO
		}
	}

	for _, entry := range uncommitted {
		entryType := func() fuse.DirentType {
			if entry.IsDir() {
				return fuse.DT_Dir
			} else {
				return fuse.DT_File
			}
		}()

		info, err := entry.Info()
		if err != nil {
			log.Printf("uncommitted: info: %v", err)
			return nil, fuse.EIO
		}

		overlayNames[entry.Name()] = struct{}{}
		overlayEntries = append(overlayEntries, fuse.Dirent{
			Inode: info.Sys().(*syscall.Stat_t).Ino, // TODO(perf): is this required?
			Name:  entry.Name(),
			Type:  entryType,
		})
	}

	for _, subdir := range d.subdirsCommitted {
		if _, overridden := overlayNames[subdir.dirBaseName]; overridden {
			continue
		}

		entries = append(entries, fuse.Dirent{
			Inode: subdir.inode,
			Name:  subdir.dirBaseName,
			Type:  fuse.DT_Dir,
		})
	}

	for _, file := range d.filesCommitted {
		if _, overridden := overlayNames[file.name]; overridden {
			continue
		}

		entries = append(entries, fuse.Dirent{
			Inode: file.inode,
			Name:  file.name,
			Type:  fuse.DT_File,
		})
	}

	if d.varastoClientStateHEAD != nil {
		entries = append(entries, d.varastoClientStateHEAD.dirent())
	}

	entries = append(entries, overlayEntries...)

	return entries, nil
}

// creates a file (mkdir has separate func)
func (d *CollectionDirNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	withErrorAndLog := func(err error) (fs.Node, fs.Handle, error) { // helper
		log.Printf("Create: %v", err)
		return nil, nil, fuse.EIO
	}

	log.Printf("Create %s", req.Name)

	workdirPath := d.workdirPath(req.Name)

	if err := os.MkdirAll(filepath.Dir(workdirPath), 0755); err != nil {
		return withErrorAndLog(err)
	}

	// TODO: this doesn't yet work for subdirs
	createdFileHandle, err := os.Create(workdirPath)
	if err != nil {
		return withErrorAndLog(err)
	}

	if err := createdFileHandle.Close(); err != nil {
		return withErrorAndLog(err)
	}

	newFile := d.overlay.getOrCreate(workdirPath)

	openResp := fuse.OpenResponse{}
	handle, err := newFile.Open(ctx, &fuse.OpenRequest{Flags: req.Flags}, &openResp)
	if err != nil {
		return nil, nil, err
	}

	resp.Handle = openResp.Handle

	return newFile, handle, nil
}

func (d *CollectionDirNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	log.Printf("mkdir %s", req.Name)

	newDirPath := d.workdirPath(req.Name)

	if err := os.MkdirAll(filepath.Dir(newDirPath), 0755); err != nil { // make sure workdir hierarchy exists
		log.Printf("MkdirAll: %v", err)
		return nil, fuse.EIO
	}

	if err := os.Mkdir(newDirPath, 0755); err != nil {
		log.Printf("mkdir: %v", err)
		return nil, fuse.EIO
	}

	stat, err := os.Stat(newDirPath)
	if err != nil {
		return nil, err
	}

	return d.overlay.getOrCreateDir(newDirPath, func() *CollectionDirNode {
		return newCollectionDirNode(
			d.collection,
			filepath.Join(d.pathFromRoot(), req.Name),
			stat.Sys().(*syscall.Stat_t).Ino,
			[]*CollectionDirNodeFile{},
			[]*CollectionDirNode{},
			d.overlay,
		)
	}), nil
}

func (d *CollectionDirNode) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	log.Printf("Remove %s", req.Name)

	if req.Dir { // rmdir not implemented
		return fuse.EPERM
	}

	for _, file := range d.filesCommitted {
		if file.name == req.Name {
			// not supported yet (need "whiteout" files)
			log.Printf("tried to delete already committed file %s", req.Name)
			return fuse.EPERM
		}
	}

	// assume remove is for uncommitted file (still don't know if it exists yet)
	err := os.Remove(d.workdirPath(req.Name))
	switch {
	case err == nil:
		return nil
	case os.IsNotExist(err):
		return fuse.ENOENT
	default: // unexpected
		log.Printf("Remove unexpected err: %v", err)
		return fuse.EIO
	}
}

func (d *CollectionDirNode) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	log.Printf("Rename %s -> %s newdir=%d", req.OldName, req.NewName, req.NewDir)

	for _, file := range d.filesCommitted {
		if file.name == req.OldName {
			// not supported yet
			log.Printf("tried to delete already committed file %s", req.OldName)
			return fuse.EPERM
		}
	}

	targetDir, ok := newDir.(*CollectionDirNode)
	if !ok {
		return fuse.EIO
	}

	oldPath := d.workdirPath(req.OldName)
	newPath := targetDir.workdirPath(req.NewName)
	err := os.Rename(oldPath, newPath)
	switch {
	case err == nil:
		d.overlay.move(
			oldPath,
			newPath,
			filepath.Join(d.pathFromRoot(), req.OldName),
			filepath.Join(targetDir.pathFromRoot(), req.NewName),
		)

		if req.OldName == stoclient.LocalStatefile+".part" && req.NewName == stoclient.LocalStatefile && d.isRoot() {
			content, err := os.ReadFile(newPath)
			if err != nil {
				log.Printf("Rename: read saved state: %v", err)
				return fuse.EIO
			}

			if err := os.RemoveAll(d.workdirPath("")); err != nil {
				log.Printf("Rename: clear workdir: %v", err)
				return fuse.EIO
			}

			if d.varastoClientStateHEAD != nil {
				d.varastoClientStateHEAD.replaceContent(content)
			}
			if collectionByIDQuerySingleton != nil {
				collectionByIDQuerySingleton.forgetCollection(ctx, d.collection.ID)
			}
		}
		return nil
	case os.IsNotExist(err):
		return fuse.ENOENT
	default: // unexpected
		log.Printf("Rename unexpected err: %v", err)
		return fuse.EIO
	}
}

func (d *CollectionDirNode) workdirPath(name string) string {
	// for collection root *fullPathFromRoot* is "." => has no effect thus ignored (= good in that case)
	return filepath.Join(home, ".local/varasto-work", d.collection.ID, d.pathFromRoot(), name)
}

func (d *CollectionDirNode) isRoot() bool {
	return d.pathFromRoot() == "."
}

func (d *CollectionDirNode) pathFromRoot() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fullPathFromRoot
}

func (d *CollectionDirNode) moveTo(fullPathFromRoot string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.fullPathFromRoot = fullPathFromRoot
	d.dirBaseName = filepath.Base(fullPathFromRoot)
}

func NewCollectionDirNodeFile(
	name string,
	inode uint64,
	file stotypes.File,
	collectionID string,
	srv *FsServer,
) *CollectionDirNodeFile {
	return &CollectionDirNodeFile{inode, file, name, collectionID, srv}
}

var _ interface {
	fs.Node
	fs.HandleReader
} = (*CollectionDirNodeFile)(nil)

type CollectionDirNodeFile struct {
	inode        uint64
	file         stotypes.File
	name         string
	collectionID string
	srv          *FsServer
}

func (f CollectionDirNodeFile) Attr(ctx context.Context, attrs *fuse.Attr) error {
	attrs.Inode = f.inode
	attrs.Mode = 0444
	attrs.Mtime = f.file.Modified
	attrs.Size = uint64(f.file.Size)

	return nil
}

func (f CollectionDirNodeFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// nothing to offer, already past file limit
	if req.Offset >= f.file.Size {
		return errors.New("offset past EOF")
	}

	for _, alignedRead := range alignReads(req.Offset, min(int64(req.Size), f.file.Size-req.Offset)) {
		blobRef, err := stotypes.BlobRefFromHex(f.file.BlobRefs[alignedRead.blobIdx])
		if err != nil {
			return err
		}

		bd, err := f.srv.blobCache.Get(ctx, *blobRef, f.collectionID)
		if err != nil {
			return err
		}

		resp.Data = append(resp.Data, bd.Data[alignedRead.offsetInBlob:int64(alignedRead.offsetInBlob)+alignedRead.lenInBlob]...)
	}

	return nil
}

// represents a changed file (from perspective of HEAD commit) present in "overlay" workdir
type changedFileInWorkdir struct {
	mu              sync.RWMutex
	backingFilePath string
}

type overlayFiles struct {
	mu    sync.Mutex
	files map[string]*changedFileInWorkdir
	dirs  map[string]*CollectionDirNode
}

func newOverlayFiles() *overlayFiles {
	return &overlayFiles{
		files: map[string]*changedFileInWorkdir{},
		dirs:  map[string]*CollectionDirNode{},
	}
}

func (o *overlayFiles) getOrCreateDir(path string, create func() *CollectionDirNode) *CollectionDirNode {
	o.mu.Lock()
	defer o.mu.Unlock()

	if dir, found := o.dirs[path]; found {
		return dir
	}

	dir := create()
	o.dirs[path] = dir
	return dir
}

func (o *overlayFiles) getOrCreate(path string) *changedFileInWorkdir {
	o.mu.Lock()
	defer o.mu.Unlock()

	if file, found := o.files[path]; found {
		return file
	}

	file := &changedFileInWorkdir{backingFilePath: path}
	o.files[path] = file
	return file
}

func (o *overlayFiles) move(oldPath string, newPath string, oldFullPath string, newFullPath string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldPathWithSeparator := oldPath + string(filepath.Separator)
	oldFullPathWithSeparator := oldFullPath + string(filepath.Separator)
	for path, file := range o.files {
		var movedPath string
		switch {
		case path == oldPath:
			movedPath = newPath
		case strings.HasPrefix(path, oldPathWithSeparator):
			movedPath = filepath.Join(newPath, strings.TrimPrefix(path, oldPathWithSeparator))
		default:
			continue
		}

		file.mu.Lock()
		file.backingFilePath = movedPath
		file.mu.Unlock()

		delete(o.files, path)
		o.files[movedPath] = file
	}

	for path, dir := range o.dirs {
		var movedPath string
		var movedFullPath string
		switch {
		case path == oldPath:
			movedPath = newPath
			movedFullPath = newFullPath
		case strings.HasPrefix(path, oldPathWithSeparator):
			movedPath = filepath.Join(newPath, strings.TrimPrefix(path, oldPathWithSeparator))
			movedFullPath = filepath.Join(newFullPath, strings.TrimPrefix(dir.pathFromRoot(), oldFullPathWithSeparator))
		default:
			continue
		}

		dir.moveTo(movedFullPath)
		delete(o.dirs, path)
		o.dirs[movedPath] = dir
	}
}

func (a *changedFileInWorkdir) path() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.backingFilePath
}

var _ interface {
	fs.Node
	fs.NodeOpener
	fs.NodeSetattrer

	// xattrs

	fs.NodeGetxattrer
	fs.NodeSetxattrer
	fs.NodeListxattrer
	fs.NodeRemovexattrer
} = (*changedFileInWorkdir)(nil)

func (a *changedFileInWorkdir) Attr(ctx context.Context, attrs *fuse.Attr) error {
	stat, err := os.Stat(a.path())
	if err != nil {
		return err
	}

	attrs.Inode = stat.Sys().(*syscall.Stat_t).Ino
	attrs.Mode = stat.Mode()
	attrs.Mtime = stat.ModTime()
	attrs.Size = uint64(stat.Size())

	return nil
}

func (a *changedFileInWorkdir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	valid := func(item fuse.SetattrValid) bool { // helper
		return req.Valid&item != 0
	}

	if valid(fuse.SetattrMode) {
		if err := os.Chmod(a.path(), req.Mode); err != nil {
			log.Printf("Setattr: chmod: %v", err)
			return fuse.EIO
		}
	}

	if valid(fuse.SetattrSize) {
		if req.Size > 1<<63-1 {
			return fuse.Errno(syscall.EINVAL)
		}

		if err := os.Truncate(a.path(), int64(req.Size)); err != nil {
			log.Printf("Setattr: truncate: %v", err)
			return fuse.EIO
		}
	}

	if valid(fuse.SetattrAtime) || valid(fuse.SetattrMtime) {
		existing, err := os.Stat(a.path())
		if err != nil {
			log.Printf("Setattr: stat: %v", err)
			return fuse.EIO
		}

		existingOrNew := func(existing time.Time, new_ time.Time, newValid bool) time.Time {
			if newValid {
				return new_
			} else {
				return existing
			}
		}

		// some platforms don't support fetching access time.
		existingATime := accessTimeFromStatt(existing.Sys().(*syscall.Stat_t), existing.ModTime())

		// TODO: use herbis times

		if err := os.Chtimes(
			a.path(),
			existingOrNew(existingATime, req.Atime, valid(fuse.SetattrAtime)),
			existingOrNew(existing.ModTime(), req.Mtime, valid(fuse.SetattrMtime)),
		); err != nil {
			log.Printf("Setattr: chtimes: %v", err)
			return fuse.EIO
		}
	}

	return nil
}

func (a *changedFileInWorkdir) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	path := a.path()
	log.Printf("open %s", path)
	hdl, err := os.OpenFile(path, int(req.Flags), 0700)
	if err != nil {
		return nil, err
	}
	resp.Handle = 666 // FIXME: WTF does this work?

	return &changedFileInWorkdirHandle{hdl}, nil
}

/*
To test

$ apt install -y attr
$ setfattr -n user.foo -v hihi foobar.txt
*/
func (a *changedFileInWorkdir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	path := a.path()
	log.Printf("Listxattr %s", path)

	attrs, err := xattr.List(path)
	if err != nil {
		log.Printf("Listxattr unexpected err: %v", err)
		return fuse.EIO
	}

	for _, attr := range attrs {
		resp.Append(attr)
	}

	return nil
}

func (a *changedFileInWorkdir) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	path := a.path()
	log.Printf("Setxattr %s %s", path, req.Name)

	if err := xattr.Set(path, req.Name, req.Xattr); err != nil {
		log.Printf("Setxattr unexpected err: %v", err)
		return fuse.EIO
	}

	return nil
}

func (a *changedFileInWorkdir) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	path := a.path()
	log.Printf("Removexattr %s %s", path, req.Name)

	err := xattr.Remove(path, req.Name)
	switch {
	case err == nil:
		return nil
	case isXattrNotExist(err):
		return fuse.ErrNoXattr
	default:
		log.Printf("Removexattr unexpected err: %v", err)
		return fuse.EIO
	}
}

func (a *changedFileInWorkdir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	// log.Printf("Getxattr %s %s", a.backingFilePath, req.Name)

	val, err := xattr.Get(a.path(), req.Name)
	switch {
	case err == nil:
		resp.Xattr = val
		return nil
	case isXattrNotExist(err):
		return fuse.ErrNoXattr
	default:
		log.Printf("Getxattr unexpected err: %v", err)
		return fuse.EIO
	}
}

// represents *changedFileInWorkdir* opened, ready for reading and writing
type changedFileInWorkdirHandle struct {
	file *os.File
}

var _ interface {
	fs.HandleReader
	fs.HandleWriter
	fs.HandleFlusher
	fs.HandleReleaser
} = (*changedFileInWorkdirHandle)(nil)

func (a *changedFileInWorkdirHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	resp.Data = make([]byte, req.Size)
	n, err := a.file.ReadAt(resp.Data, req.Offset)
	resp.Data = resp.Data[:n]
	if err == io.EOF { // happens at least with empty files (given read buffer larger than file has content)
		return nil
	} else {
		return err
	}
}

func (a *changedFileInWorkdirHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	n, err := a.file.WriteAt(req.Data, req.Offset)
	resp.Size = n
	if err != nil {
		log.Printf("write error: %v", err)
		return fuse.EIO
	}

	return nil
}

func (a *changedFileInWorkdirHandle) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	// log.Printf("got flush for %d (%s)", req.Handle, a.file.Name())
	return nil
}

func (a *changedFileInWorkdirHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return a.file.Close()
}

func isXattrNotExist(err error) bool {
	// these don't work:
	// case err==xattr.ENOATTR:
	// case os.IsNotExist(err):

	// FIXME: horrible, terrible
	if xattrErr, ok := err.(*xattr.Error); ok && strings.Contains(xattrErr.Err.Error(), "no data available") {
		return true
	} else {
		return false
	}
}
