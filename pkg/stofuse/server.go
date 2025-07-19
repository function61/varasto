package stofuse

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/retry"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

// guaranteed non-null
// FIXME: dirty
var home = func() string {
	val := os.Getenv("HOME")
	if val == "" {
		panic("HOME ENV not set")
	}

	return val
}()

func fuseServe(
	ctx context.Context,
	sigs *sigFabric,
	conf stoclient.ClientConfig,
	unmountFirst bool,
	logl *logex.Leveled,
) error {
	if conf.FuseMountPath == "" {
		return errors.New("FuseMountPath not set")
	}

	if err := makeMountpointIfRequired(conf.FuseMountPath); err != nil {
		return err
	}

	// we can't do this without the branch because if path is not mounted, it yields an error
	// and I would feel uncomfortable trying to detect "not mounted" error vs "any other error"
	if unmountFirst {
		// if previous process dies before successfull unmount, this will unmount it without root privileges
		if err := fuse.Unmount(conf.FuseMountPath); err != nil {
			return err
		}
	}

	// there's fuse.AllowRoot() available but it doesn't seem to be supported by "$ fusermount" binary.
	// https://github.com/bazil/fuse/issues/144

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

	srv := NewFsServer(conf.Client(), conf.FuseMountPath, logl)

	byIDDir := NewByIDDir(srv)

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case <-sigs.unmountAll:
				byIDDir.ForgetDirs()
			}
		}
	}()

	go func() {
		<-ctx.Done()

		tryUnmount := func(ctx context.Context) error {
			// "Instead of sending a SIGINT to the process, you should unmount the filesystem
			// That will cause the serve loop to exit, and your process can exit that way."
			// https://github.com/bazil/fuse/issues/6
			return fuse.Unmount(conf.FuseMountPath)
		}

		ctx, cancel := context.WithTimeout(context.TODO(), 45*time.Second)
		defer cancel()

		// retrying because unmount will fail if any process is accessing the mount
		if err := retry.Retry(ctx, tryUnmount, retry.DefaultBackoff(), func(err error) {
			logl.Error.Printf("tryUnmount: %v", err)
		}); err != nil {
			panic(err)
		}

		// this succeeding will unblock <-fuseConn.Ready
	}()

	dirByID := NewDirByIDQuery(srv)

	varastoFsRoot, err := NewVarastoFSRoot([]stoEntry{
		newStoEntry(fuse.Dirent{
			Name:  "id",
			Type:  fuse.DT_Dir,
			Inode: byIDDir.inode,
		}, byIDDir),
		newStoEntry(fuse.Dirent{
			Name:  "dir",
			Type:  fuse.DT_Dir,
			Inode: dirByID.inode,
		}, dirByID),
		newStoSymlink(srv.AbsoluteSymlinkUnderFUSEMount("dir", stoservertypes.RootFolderId)).MakeStoEntry("browse"),
	}, srv)
	if err != nil {
		return err
	}

	logl.Info.Println("VarastoFS server started")

	if err := fs.Serve(fuseConn, varastoFsRoot); err != nil {
		return err
	}

	// check if the mount process has an error to report
	// blocks as long as server is up
	<-fuseConn.Ready
	if err := fuseConn.MountError; err != nil {
		return err
	}

	return nil
}

type FsServer struct {
	client        *stoclient.Client
	blobCache     *BlobCache
	fuseMountPath string // needed for making absolute symlinks
	logl          *logex.Leveled
}

func NewFsServer(client *stoclient.Client, fuseMountPath string, logl *logex.Leveled) *FsServer {
	return &FsServer{
		client:        client,
		blobCache:     NewBlobCache(client.Config(), logl),
		fuseMountPath: fuseMountPath,
		logl:          logl,
	}
}

// absolute paths are better because relative ones don't seem to work over multiple symlink levels
// (or something else level-dependent breaks on them..)
func (f *FsServer) AbsoluteSymlinkUnderFUSEMount(component0 string, components ...string) string {
	allComponents := append([]string{f.fuseMountPath, component0}, components...)
	return filepath.Join(allComponents...)
}

// TODO: make the root return "backup exclude" xattr
type VarastoFSRoot struct {
	srv     *FsServer
	inode   uint64
	entries []stoEntry
}

func NewVarastoFSRoot(entries []stoEntry, srv *FsServer) (*VarastoFSRoot, error) {
	return &VarastoFSRoot{srv, nextInode(), entries}, nil
}

// implements fs.FS
func (v *VarastoFSRoot) Root() (fs.Node, error) {
	return v, nil
}

func (v *VarastoFSRoot) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = v.inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (v *VarastoFSRoot) Lookup(ctx context.Context, name string) (fs.Node, error) {
	for _, item := range v.entries {
		if item.dirent.Name == name {
			return item.node, nil
		}
	}

	return nil, fuse.ENOENT
}

func (v *VarastoFSRoot) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dentries := []fuse.Dirent{}
	for _, entry := range v.entries {
		dentries = append(dentries, entry.dirent)
	}

	return dentries, nil
}
