package blobdriver

import (
	"github.com/function61/bup/pkg/buptypes"
	"io"
)

type Driver interface {
	Store(ref buptypes.BlobRef, rd io.Reader) (int64, error)

	// if chunk is not found, error must report os.IsNotExist(err) == true
	Fetch(ref buptypes.BlobRef) (io.ReadCloser, error)
}
