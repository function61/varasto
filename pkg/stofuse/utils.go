package stofuse

import (
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
	"regexp"
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

func min(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}
