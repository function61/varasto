package stofuse

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
)

var reservedInodeCounter = uint64(0)

func nextInode() uint64 {
	reservedInodeCounter++
	return reservedInodeCounter
}

type alignedBlobRead struct {
	blobIdx      int
	offsetInBlob int
	lenInBlob    int64
}

// aligns file's reads within blob boundaries
func alignReads(offsetInFile int64, readLen int64) []alignedBlobRead {
	blobIdx, offsetInBlob := stoclient.BlobIdxFromOffset(offsetInFile)

	// simplest, general case
	if offsetInBlob+readLen <= stotypes.BlobSize {
		return []alignedBlobRead{
			{blobIdx: blobIdx, offsetInBlob: int(offsetInBlob), lenInBlob: readLen},
		}
	}

	firstRead := alignedBlobRead{blobIdx: blobIdx, offsetInBlob: int(offsetInBlob), lenInBlob: stotypes.BlobSize - offsetInBlob}
	readLen -= firstRead.lenInBlob

	additionalReads := []alignedBlobRead{}

	for readLen > 0 {
		blobIdx++
		readLenForBlob := min(readLen, stotypes.BlobSize)
		additionalReads = append(additionalReads, alignedBlobRead{blobIdx: blobIdx, offsetInBlob: 0, lenInBlob: readLenForBlob})

		readLen -= readLenForBlob
	}

	return append([]alignedBlobRead{firstRead}, additionalReads...)
}

// https://serverfault.com/a/650041
// \ / : * ? " < > |
var fsWindowsUnsafeRe = regexp.MustCompile("[\\\\/:*?\"<>|]")

func mkFsSafe(input string) string {
	return fsWindowsUnsafeRe.ReplaceAllString(input, "_")
}

func makeMountpointIfRequired(mountpoint string) error {
	mountpointExists, err := fileexists.Exists(mountpoint)
	if err != nil {
		return err
	}
	if !mountpointExists {
		if err := os.MkdirAll(mountpoint, 0755); err != nil {
			return fmt.Errorf("failed making mount point: %w", err)
		}
	}

	return nil
}

func createHeadState(coll *stotypes.Collection) *staticFile {
	headStateJSON := &bytes.Buffer{}

	if err := jsonfile.Marshal(headStateJSON, stoclient.BupManifest{
		ChangesetID: func() string {
			lastChangesetIdx := len(coll.Changesets) - 1
			if lastChangesetIdx != -1 {
				return coll.Changesets[lastChangesetIdx].ID
			} else {
				return ""
			}
		}(),
		Collection: *coll,
	}); err != nil {
		panic(err) // not expected
	}

	return &staticFile{
		inode:   nextInode(),
		name:    stoclient.LocalStatefile,
		content: headStateJSON.Bytes(),
	}
}

// a static (readonly) file whose content we know in-memory
type staticFile struct {
	inode   uint64
	name    string
	content []byte
}

var _ interface {
	fs.Node
	fs.HandleReadAller
} = (*staticFile)(nil)

func (d *staticFile) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = d.inode
	a.Mode = 0555
	a.Size = uint64(len(d.content))
	return nil
}

func (d *staticFile) ReadAll(_ context.Context) ([]byte, error) {
	return d.content, nil
}

func (d *staticFile) dirent() fuse.Dirent {
	return fuse.Dirent{
		Inode: d.inode,
		Name:  d.name,
		Type:  fuse.DT_File,
	}
}

// TODO: use from gokit
func timespecToTime(ts syscall.Timespec) time.Time {
	//nolint:unconvert // lint thinks these are already int64 even though they're int32
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}
