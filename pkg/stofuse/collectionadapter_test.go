package stofuse

import (
	"context"
	"os"
	"testing"

	"bazil.org/fuse"
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
