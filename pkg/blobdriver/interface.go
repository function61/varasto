package blobdriver

import (
	"github.com/function61/varasto/pkg/varastotypes"
	"io"
)

type Driver interface {
	Store(ref varastotypes.BlobRef, rd io.Reader) (int64, error)

	// if chunk is not found, error must report os.IsNotExist(err) == true
	Fetch(ref varastotypes.BlobRef) (io.ReadCloser, error)

	Mountable() error
}
