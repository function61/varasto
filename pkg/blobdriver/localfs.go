package blobdriver

import (
	"fmt"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
	"io"
	"log"
	"os"
	"path/filepath"
)

func NewLocalFs(uuid string, path string, logger *log.Logger) *localFs {
	return &localFs{
		uuid: uuid,
		path: path,
		log:  logex.Levels(logex.NonNil(logger)),
	}
}

type localFs struct {
	uuid string
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

	if err := tempFileContent.Close(); err != nil { // double close is intentional
		return bytesWritten, err
	}

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

func (l *localFs) Mountable() error {
	// to ensure that we mounted correct volume, there must be a <uuid>.iogrid file in the root
	flagFilename := l.uuid + ".iogrid"

	exists, err := fileexists.Exists(filepath.Join(l.path, flagFilename))
	if err != nil {
		return err // error checking file existence
	}

	if !exists {
		return fmt.Errorf("flag file not found: %s", flagFilename)
	}

	return nil
}

func (l *localFs) getPath(ref buptypes.BlobRef) string {
	hexHash := ref.AsHex()

	// this should yield 4 096 directories as maximum (see test file for clarification)
	return filepath.Join(
		l.path,
		hexHash[0:2],
		hexHash[2:3],
		hexHash[3:]+".chunk")
}
