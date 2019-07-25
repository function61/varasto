package stoclient

import (
	"github.com/function61/varasto/pkg/stotypes"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func BlobIdxFromOffset(offset uint64) (int, int64) {
	blobIdx := int(offset / stotypes.BlobSize)
	return blobIdx, int64(offset) - (int64(blobIdx) * stotypes.BlobSize)
}
