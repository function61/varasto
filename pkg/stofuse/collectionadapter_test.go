package stofuse

import (
	"context"
	"os"
	"testing"

	"bazil.org/fuse"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
)

func TestOverlayFileNodeFollowsRename(t *testing.T) {
	workdir := t.TempDir()
	oldHome := home
	home = workdir
	defer func() { home = oldHome }()

	dir := NewCollectionDirNode(&stotypes.Collection{ID: "collection"}, ".", 1, nil, nil)
	ctx := context.Background()
	node, handle, err := dir.Create(ctx, &fuse.CreateRequest{Name: "download.crdownload", Flags: fuse.OpenReadWrite}, &fuse.CreateResponse{})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.(*changedFileInWorkdirHandle).file.Close()

	if err := dir.Rename(ctx, &fuse.RenameRequest{OldName: "download.crdownload", NewName: "download"}, dir); err != nil {
		t.Fatal(err)
	}

	if err := node.(*changedFileInWorkdir).Attr(ctx, &fuse.Attr{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir.workdirPath("download")); err != nil {
		t.Fatal(err)
	}
}
func TestOverlayFileNodeFollowsDirectoryRename(t *testing.T) {
	workdir := t.TempDir()
	oldHome := home
	home = workdir
	defer func() { home = oldHome }()

	root := NewCollectionDirNode(&stotypes.Collection{ID: "collection"}, ".", 1, nil, nil)
	if err := os.MkdirAll(root.workdirPath("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root.workdirPath("old/file"), []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	oldDir, err := root.Lookup(ctx, "old")
	if err != nil {
		t.Fatal(err)
	}
	file, err := oldDir.(*CollectionDirNode).Lookup(ctx, "file")
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(ctx, &fuse.RenameRequest{OldName: "old", NewName: "new"}, root); err != nil {
		t.Fatal(err)
	}

	if err := file.(*changedFileInWorkdir).Attr(ctx, &fuse.Attr{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root.workdirPath("new/file")); err != nil {
		t.Fatal(err)
	}

	_, handle, err := oldDir.(*CollectionDirNode).Create(ctx, &fuse.CreateRequest{Name: "newfile", Flags: fuse.OpenReadWrite}, &fuse.CreateResponse{})
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.(*changedFileInWorkdirHandle).Release(ctx, &fuse.ReleaseRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root.workdirPath("new/newfile")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root.workdirPath("old/newfile")); !os.IsNotExist(err) {
		t.Fatalf("old directory file stat error = %v, want not exist", err)
	}
}

func TestOverlayFileHandleReleaseClosesFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "overlay")
	if err != nil {
		t.Fatal(err)
	}

	handle := &changedFileInWorkdirHandle{file: file}
	if err := handle.Release(context.Background(), &fuse.ReleaseRequest{}); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err == nil {
		t.Fatal("file remained open after Release")
	}
}

func TestOverlayFileHandleUsesRequestOffsets(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "overlay")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	handle := &changedFileInWorkdirHandle{file: file}
	ctx := context.Background()
	if err := handle.Write(ctx, &fuse.WriteRequest{Offset: 3, Data: []byte("def")}, &fuse.WriteResponse{}); err != nil {
		t.Fatal(err)
	}
	if err := handle.Write(ctx, &fuse.WriteRequest{Offset: 0, Data: []byte("abc")}, &fuse.WriteResponse{}); err != nil {
		t.Fatal(err)
	}

	response := &fuse.ReadResponse{}
	if err := handle.Read(ctx, &fuse.ReadRequest{Offset: 0, Size: 6}, response); err != nil {
		t.Fatal(err)
	}
	if got, want := string(response.Data), "abcdef"; got != want {
		t.Fatalf("read = %q, want %q", got, want)
	}
}

func TestOverlayFileSetattrTruncates(t *testing.T) {
	path := t.TempDir() + "/overlay"
	if err := os.WriteFile(path, []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}

	node := &changedFileInWorkdir{backingFilePath: path}
	if err := node.Setattr(context.Background(), &fuse.SetattrRequest{
		Valid: fuse.SetattrSize,
		Size:  3,
	}, &fuse.SetattrResponse{}); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "abc"; got != want {
		t.Fatalf("contents = %q, want %q", got, want)
	}
}

func TestOverlayFileOverridesCommittedFile(t *testing.T) {
	workdir := t.TempDir()
	oldHome := home
	home = workdir
	defer func() { home = oldHome }()

	dir := NewCollectionDirNode(
		&stotypes.Collection{ID: "collection"},
		".",
		1,
		[]*CollectionDirNodeFile{{name: "download"}},
		nil,
	)
	if err := os.MkdirAll(dir.workdirPath(""), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir.workdirPath("download"), []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}

	node, err := dir.Lookup(context.Background(), "download")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := node.(*changedFileInWorkdir); !ok {
		t.Fatalf("lookup returned %T, want overlay file", node)
	}

	entries, err := dir.ReadDirAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "download" {
		t.Fatalf("directory entries = %#v, want only download", entries)
	}
}

func TestSavingStateClearsOverlayAfterAtomicRename(t *testing.T) {
	workdir := t.TempDir()
	oldHome := home
	home = workdir
	defer func() { home = oldHome }()

	dir := NewCollectionDirNode(&stotypes.Collection{ID: "collection"}, ".", 1, nil, nil)
	dir.varastoClientStateHEAD = &staticFile{name: stoclient.LocalStatefile, content: []byte("old")}
	ctx := context.Background()
	node, handle, err := dir.Create(ctx, &fuse.CreateRequest{Name: stoclient.LocalStatefile + ".part", Flags: fuse.OpenReadWrite}, &fuse.CreateResponse{})
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.(*changedFileInWorkdirHandle).Write(ctx, &fuse.WriteRequest{Data: []byte("new")}, &fuse.WriteResponse{}); err != nil {
		t.Fatal(err)
	}
	if err := handle.(*changedFileInWorkdirHandle).Release(ctx, &fuse.ReleaseRequest{}); err != nil {
		t.Fatal(err)
	}
	if err := dir.Rename(ctx, &fuse.RenameRequest{OldName: stoclient.LocalStatefile + ".part", NewName: stoclient.LocalStatefile}, dir); err != nil {
		t.Fatal(err)
	}

	contents, err := dir.varastoClientStateHEAD.ReadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "new"; got != want {
		t.Fatalf("state = %q, want %q", got, want)
	}
	if _, err := os.Stat(dir.workdirPath("")); !os.IsNotExist(err) {
		t.Fatalf("workdir stat error = %v, want not exist", err)
	}
	if _, ok := node.(*changedFileInWorkdir); !ok {
		t.Fatalf("created node = %T, want overlay file", node)
	}
}
