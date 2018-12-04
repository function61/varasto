package blobdriver

import (
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
	"io"
	"log"
	"os"
	"path/filepath"
)

func NewLocalFs(path string, logger *log.Logger) *localFs {
	return &localFs{
		path: path,
		log:  logex.Levels(logex.NonNil(logger)),
	}
}

type localFs struct {
	path string
	log  *logex.Leveled
}

func (l *localFs) Store(ref buptypes.BlobRef, content io.Reader) (int64, error) {
	finalName := l.getPath(ref)
	tempName := finalName + ".temp"

	// does not error if already exists
	if err := os.MkdirAll(filepath.Dir(finalName), 0755); err != nil {
		return 0, err
	}

	chunkExists, err := fileexists.Exists(finalName)
	if err != nil {
		return 0, err
	}

	if chunkExists {
		return 0, buptypes.ErrChunkAlreadyExists
	}

	tempFileContent, err := os.Create(tempName)
	if err != nil {
		return 0, err
	}

	success := false

	// try to ensure cleanup
	defer func() {
		tempFileContent.Close()

		if !success {
			if err := os.Remove(tempName); err != nil {
				l.log.Error.Printf("temp file %s cleanup: %s", tempName, err.Error())
			}
		}
	}()

	bytesWritten, err := io.Copy(tempFileContent, content)
	if err != nil {
		return bytesWritten, err
	}

	tempFileContent.Close()

	// rename can replace target file (there's a race condition with the file exists check),
	// but that is ok because both contents are hash-checked
	if err := os.Rename(tempName, finalName); err != nil {
		return bytesWritten, err
	}

	success = true

	return bytesWritten, nil
}

func (l *localFs) Fetch(ref buptypes.BlobRef) (io.ReadCloser, error) {
	return os.Open(l.getPath(ref))
}

func (l *localFs) getPath(ref buptypes.BlobRef) string {
	hexHash := ref.AsHex()

	return l.path + hexHash[0:2] + "/" + hexHash[2:] + ".chunk"
}
