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

// TODO: rename
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

func (c *ClientConfig) NewWorkdirLocation(path string) (*workdirLocation, error) {
	wd, err := c.NewMaybeWorkdirLocation(path)
	if err != nil {
		return wd, err
	}

	if wd == nil { // our fn contract is that it must be a Varasto workdir
		return nil, fmt.Errorf("not a Varasto workdir: %s", path)
	}

	return wd, err
}

// returns nil result without an error, if dir is not a Varasto workdir
func (c *ClientConfig) NewMaybeWorkdirLocation(path string) (*workdirLocation, error) {
	loc := &workdirLocation{
		path:         path,
		clientConfig: *c,
	}

	statefile, err := os.Open(loc.Join(localStatefile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not a Varasto workdir
		} else {
			return nil, err // some other error
		}
	}
	defer statefile.Close()

	loc.manifest = &BupManifest{}
	return loc, jsonfile.Unmarshal(statefile, loc.manifest, true)
}

/*
func NewWorkdirLocation(path string) (*workdirLocation, error) {
	clientConfig, err := ReadConfig()
	if err != nil {
		return nil, err
	}

	return clientConfig.NewWorkdirLocation(path)
}
*/

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
