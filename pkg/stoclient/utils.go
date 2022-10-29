package stoclient

import (
	"io/fs"
	"net/http"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stotypes"
)

func BlobIdxFromOffset(offset int64) (int, int64) {
	blobIdx := int(offset / stotypes.BlobSize)
	return blobIdx, offset - (int64(blobIdx) * stotypes.BlobSize)
}

func boolToStr(input bool) string {
	if input {
		return "true"
	} else {
		return "false"
	}
}

func translate404ToFSErrNotExist(err error) error {
	if err != nil {
		if ezhttp.ErrorIs(err, http.StatusNotFound) {
			return fs.ErrNotExist
		} else { // some other error
			return err
		}
	}

	return nil // no error
}
