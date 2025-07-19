package stodiskaccess

import (
	"io"

	"github.com/function61/varasto/pkg/stotypes"
)

type BlobMeta struct {
	Ref             stotypes.BlobRef
	RealSize        int32
	SizeOnDisk      int32 // after optional compression
	IsCompressed    bool
	EncryptionKeyID string
	EncryptionKey   []byte // this is set when read from QueryBlobMetadata(), but not when given to WriteBlobCreated()
	ExpectedCrc32   []byte
}

type MetadataStore interface {
	// returns os.ErrNotExist if ref does not exist
	QueryBlobMetadata(ref stotypes.BlobRef, encryptionKeys []stotypes.KeyEnvelope) (*BlobMeta, error)
	QueryBlobCrc32(ref stotypes.BlobRef) ([]byte, error)
	QueryBlobExists(ref stotypes.BlobRef) (bool, error)
	QueryCollectionEncryptionKeyForNewBlobs(collID string) (string, []byte, error)
	WriteBlobCreated(meta *BlobMeta, volumeID int) error
	WriteBlobReplicated(ref stotypes.BlobRef, volumeID int) error
}

type readCloseWrapper struct {
	io.Reader
	closer io.Closer
}

func (r *readCloseWrapper) Close() error {
	return r.closer.Close()
}
