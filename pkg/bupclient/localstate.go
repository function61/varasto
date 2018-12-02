package bupclient

import (
	"encoding/json"
	"errors"
	"github.com/function61/bup/pkg/buptypes"
	"os"
	"path/filepath"
)

const (
	localStatefile = ".bup"
)

type BupManifest struct {
	ChangesetId string              `json:"changeset_id"`
	Collection  buptypes.Collection `json:"collection"` // snapshot at time of server fetch
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
	file, err := os.Create(w.Join(localStatefile))
	if err != nil {
		return err
	}
	defer file.Close()

	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetIndent("", "  ")
	return jsonEncoder.Encode(w.manifest)
}

func NewWorkdirLocation(path string) (*workdirLocation, error) {
	clientConfig, err := readConfig()
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
			return nil, errors.New("current dir not a bup workdir")
		}

		return nil, err // some other error
	}
	defer statefile.Close()

	manifest := &BupManifest{}

	dec := json.NewDecoder(statefile)
	dec.DisallowUnknownFields()
	if err := dec.Decode(manifest); err != nil {
		return nil, err
	}

	loc.manifest = manifest

	return loc, nil
}
