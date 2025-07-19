// Computes the state of collection at an exact revision. The revision's parent DAG is
// traversed back to the root to compute all the deltas.
package stateresolver

import (
	"errors"
	"sort"

	"github.com/function61/varasto/pkg/stotypes"
)

// keyed by file path
type fileMap map[string]stotypes.File

type StateAt struct {
	ChangesetID string
	files       fileMap
}

func (s *StateAt) Files() fileMap {
	files := fileMap{}

	// makes a clone
	for key, value := range s.files {
		files[key] = value
	}

	return files
}

// List of files present at this revision
func (s *StateAt) FileList() []stotypes.File {
	files := []stotypes.File{}

	for _, file := range s.files {
		files = append(files, file)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files
}

func ComputeStateAtHead(c stotypes.Collection) (*StateAt, error) {
	return ComputeStateAt(c, c.Head)
}

func ComputeStateAt(c stotypes.Collection, changesetID string) (*StateAt, error) {
	state := &StateAt{
		ChangesetID: changesetID,
		files:       fileMap{},
	}

	// initial state is always empty
	if changesetID == stotypes.NoParentID {
		return state, nil
	}

	ch := findChangesetByID(c, changesetID)
	if ch == nil {
		return nil, errors.New("changeset not found")
	}

	parents := []*stotypes.CollectionChangeset{ch}

	curr := ch

	// because this is a DAG, our only option is to traverse from newest to oldest
	// direction. we'll have to do processing in reverse order though (oldest to newest)
	for curr.Parent != stotypes.NoParentID {
		parent := findChangesetByID(c, curr.Parent)
		if parent == nil {
			return nil, errors.New("parent not found")
		}

		parents = append(parents, parent)

		curr = parent
	}

	// process in reverse order (from oldest to newest) because otherwise resulting
	// state would be borked
	for i := len(parents) - 1; i >= 0; i-- {
		parent := parents[i]

		for _, add := range parent.FilesCreated {
			state.files[add.Path] = add
		}
		for _, remove := range parent.FilesDeleted {
			delete(state.files, remove)
		}
		for _, update := range parent.FilesUpdated {
			state.files[update.Path] = update
		}
	}

	return state, nil
}

func findChangesetByID(c stotypes.Collection, id string) *stotypes.CollectionChangeset {
	for _, changeset := range c.Changesets {
		if changeset.ID == id {
			return &changeset
		}
	}

	return nil
}
