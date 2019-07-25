package localfsblobstore

import (
	"fmt"
	"github.com/function61/gokit/atomicfilewrite"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"io"
	"log"
	"os"
	"path/filepath"
)

func New(uuid string, path string, logger *log.Logger) *localFs {
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

func (l *localFs) RawStore(ref stotypes.BlobRef, content io.Reader) error {
	filename := l.getPath(ref)

	// does not error if already exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	// TODO: this exists check is not strictly necessary, since the file
	// is written in atomic manner
	chunkExists, err := fileexists.Exists(filename)
	if err != nil {
		return err
	}

	if chunkExists {
		return stotypes.ErrChunkAlreadyExists
	}

	return atomicfilewrite.Write(filename, func(writer io.Writer) error {
		_, err := io.Copy(writer, content)
		return err
	})
}

func (l *localFs) RawFetch(ref stotypes.BlobRef) (io.ReadCloser, error) {
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

func (l *localFs) getPath(ref stotypes.BlobRef) string {
	hexHash := ref.AsHex()

	// this should yield 4 096 directories as maximum (see test file for clarification)
	return filepath.Join(
		l.path,
		hexHash[0:2],
		hexHash[2:3],
		hexHash[3:]+".chunk")
}
