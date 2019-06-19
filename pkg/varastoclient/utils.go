package varastoclient

import (
	"github.com/function61/varasto/pkg/varastotypes"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func BlobIdxFromOffset(offset uint64) (int, int64) {
	blobIdx := int(offset / varastotypes.BlobSize)
	return blobIdx, int64(offset) - (int64(blobIdx) * varastotypes.BlobSize)
}
