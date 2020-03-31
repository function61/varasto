package stoclient

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/stotypes"
)

const (
	localStatefile = ".varasto"
)

type BupManifest struct {
	ChangesetId string              `json:"changeset_id"`
	Collection  stotypes.Collection `json:"collection"` // snapshot at time of server fetch
}

// TODO: rename to some kind of context
type workdirLocation struct {
	path         string
	clientConfig ClientConfig
	manifest     *BupManifest
}

func (w *workdirLocation) Join(comp string) string {
	return filepath.Join(w.path, comp)
}

func (w *workdirLocation) SaveToDisk() error {
	return jsonfile.Write(w.Join(localStatefile), w.manifest)
}

func NewWorkdirLocation(path string) (*workdirLocation, error) {
	clientConfig, err := ReadConfig()
	if err != nil {
		return nil, err
	}

	loc := &workdirLocation{
		path:         path,
		clientConfig: *clientConfig,
	}

	statefile, err := os.Open(loc.Join(localStatefile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not a Varasto workdir: %s", loc.path)
		}

		return nil, err // some other error
	}
	defer statefile.Close()

	loc.manifest = &BupManifest{}
	return loc, jsonfile.Unmarshal(statefile, loc.manifest, true)
}

func statefileExists(path string) (bool, error) {
	// init this in "hack mode" (i.e. statefile not being read to memory). as soon as we
	// manage to write the statefile to disk, use normal procedure to init wd
	halfBakedWd := &workdirLocation{
		path: path,
	}

	return fileexists.Exists(halfBakedWd.Join(localStatefile))
}

func assertStatefileNotExists(path string) error {
	if exists, err := statefileExists(path); err != nil || exists {
		if err != nil { // error doing the check
			return err
		}

		return fmt.Errorf("%s already exists in %s - adopting would be dangerous", localStatefile, path)
	}

	return nil
}
