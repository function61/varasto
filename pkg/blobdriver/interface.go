package blobdriver

import (
	"github.com/function61/varasto/pkg/varastotypes"
	"io"
)

type Driver interface {
	// backing store must be idempotent, i.e. writing same blob again must not change outcome.
	// write also must be atomic. Fetch() must not return anything before store is completed succesfully.
	RawStore(ref varastotypes.BlobRef, content io.Reader) error

	// raw = driver doesn't do any encryption/compression/integrity verifications,
	//       they are done at a higher level.
	// if blob is not found, error must report os.IsNotExist(err) == true
	RawFetch(ref varastotypes.BlobRef) (io.ReadCloser, error)

	Mountable() error
}
