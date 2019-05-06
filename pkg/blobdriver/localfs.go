package blobdriver

import (
	"fmt"
	"github.com/function61/gokit/atomicfilewrite"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/varastotypes"
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

func (l *localFs) Store(ref varastotypes.BlobRef, content io.Reader) (int64, error) {
	filename := l.getPath(ref)

	// does not error if already exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return 0, err
	}

	chunkExists, err := fileexists.Exists(filename)
	if err != nil {
		return 0, err
	}

	if chunkExists {
		return 0, varastotypes.ErrChunkAlreadyExists
	}

	bytesWritten := int64(0)
	err = atomicfilewrite.Write(filename, func(writer io.Writer) error {
		bytesWritten, err = io.Copy(writer, content)
		return err
	})

	return bytesWritten, err
}

func (l *localFs) Fetch(ref varastotypes.BlobRef) (io.ReadCloser, error) {
	return os.Open(l.getPath(ref))
}

func (l *localFs) Mountable() error {
	// to ensure that we mounted correct volume, there must be a flag file in the root.
	// without this check, we could accidentally mount the wrong volume and that would be bad.
	flagFilename := "varasto-" + l.uuid + ".json"

	exists, err := fileexists.Exists(filepath.Join(l.path, flagFilename))
	if err != nil {
		return err // error checking file existence
	}

	if !exists {
		return fmt.Errorf("flag file not found: %s", flagFilename)
	}

	return nil
}

func (l *localFs) getPath(ref varastotypes.BlobRef) string {
	hexHash := ref.AsHex()

	// this should yield 4 096 directories as maximum (see test file for clarification)
	return filepath.Join(
		l.path,
		hexHash[0:2],
		hexHash[2:3],
		hexHash[3:]+".chunk")
}
