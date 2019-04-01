package varastoclient

import (
	"fmt"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/varastotypes"
	"os"
	"path/filepath"
)

const (
	localStatefile = ".varasto"
)

type BupManifest struct {
	ChangesetId string                  `json:"changeset_id"`
	Collection  varastotypes.Collection `json:"collection"` // snapshot at time of server fetch
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
	if err := jsonfile.Unmarshal(statefile, loc.manifest, true); err != nil {
		return nil, err
	}

	return loc, nil
}
