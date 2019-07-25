package stodiskaccess

import (
	"github.com/function61/varasto/pkg/stotypes"
	"io"
)

type BlobMeta struct {
	Ref                 stotypes.BlobRef
	RealSize            int32
	SizeOnDisk          int32 // after optional compression
	IsCompressed        bool
	EncryptionKeyOfColl string
	EncryptionKey       []byte // this is set when read from QueryBlobMetadata(), but not when given to WriteBlobCreated()
	ExpectedCrc32       []byte
}

type MetadataStore interface {
	// returns os.ErrNotExist if ref does not exist
	QueryBlobMetadata(ref stotypes.BlobRef) (*BlobMeta, error)
	QueryCollectionEncryptionKey(collId string) ([]byte, error)
	WriteBlobCreated(meta *BlobMeta, volumeId int) error
	WriteBlobReplicated(meta *BlobMeta, volumeId int) error
}

type readCloseWrapper struct {
	io.Reader
	closer io.Closer
}

func (r *readCloseWrapper) Close() error {
	return r.closer.Close()
}
