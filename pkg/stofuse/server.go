package stofuse

import (
	"context"
	"errors"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/retry"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stoclient"
)

func fuseServe(sigs *sigFabric, conf stoclient.ClientConfig, unmountFirst bool, stop *stopper.Stopper, logl *logex.Leveled) error {
	defer stop.Done()

	if conf.FuseMountPath == "" {
		return errors.New("FuseMountPath not set")
	}

	// we can't do this without the branch because if path is not mounted, it yields an error
	// and I would feel uncomfortable trying to detect "not mounted" error vs "any other error"
	if unmountFirst {
		// if previous process dies before successfull unmount, this will unmount it without root privileges
		if err := fuse.Unmount(conf.FuseMountPath); err != nil {
			return err
		}
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

	srv := NewFsServer(conf, logl)

	byIdDir := NewByIdDir(srv)

	go func() {
		for {
			select {
			case <-stop.Signal:
				break
			case <-sigs.unmountAll:
				byIdDir.ForgetDirs()
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

		ctx, cancel := context.WithTimeout(context.TODO(), 45*time.Second)
		defer cancel()

		// retrying because unmount will fail if any process is accessing the mount
		if err := retry.Retry(ctx, tryUnmount, retry.DefaultBackoff(), func(err error) {
			logl.Error.Printf("tryUnmount: %v", err)
		}); err != nil {
			panic(err)
		}
	}()

	varastoFsRoot, err := NewVarastoFSRoot(byIdDir, srv)
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
		logl.Error.Fatal(err)
	}

	return nil
}

type FsServer struct {
	clientConfig stoclient.ClientConfig
	blobCache    *BlobCache
	logl         *logex.Leveled
}

func NewFsServer(clientConfig stoclient.ClientConfig, logl *logex.Leveled) *FsServer {
	return &FsServer{
		clientConfig: clientConfig,
		blobCache:    NewBlobCache(clientConfig, logl),
		logl:         logl,
	}
}

type VarastoFSRoot struct {
	srv     *FsServer
	inode   uint64
	byIdDir *byIdDir
}

func NewVarastoFSRoot(byIdDir *byIdDir, srv *FsServer) (*VarastoFSRoot, error) {
	return &VarastoFSRoot{srv, nextInode(), byIdDir}, nil
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
	switch name {
	case "id":
		return v.byIdDir, nil
	default:
		return nil, fuse.ENOENT
	}
}

func (v *VarastoFSRoot) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{
		{
			Name:  "id",
			Type:  fuse.DT_Dir,
			Inode: v.byIdDir.inode,
		},
	}, nil
}
