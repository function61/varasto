package stateresolver

import (
	"errors"
	"github.com/function61/bup/pkg/buptypes"
	"sort"
)

type fileMap map[string]buptypes.File

type StateAt struct {
	ChangesetId string
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

func (s *StateAt) FileList() []buptypes.File {
	files := []buptypes.File{}

	for _, file := range s.files {
		files = append(files, file)
	}

	sort.Sort(byPath(files))

	return files
}

func ComputeStateAt(c buptypes.Collection, changesetId string) (*StateAt, error) {
	state := &StateAt{
		ChangesetId: changesetId,
		files:       fileMap{},
	}

	// initial state is always empty
	if changesetId == buptypes.NoParentId {
		return state, nil
	}

	ch := findChangesetById(c, changesetId)
	if ch == nil {
		return nil, errors.New("changeset not found")
	}

	parents := []*buptypes.CollectionChangeset{ch}

	curr := ch

	for curr.Parent != buptypes.NoParentId {
		parent := findChangesetById(c, curr.Parent)
		if parent == nil {
			return nil, errors.New("parent not found")
		}

		parents = append(parents, parent)

		curr = parent
	}

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

func findChangesetById(c buptypes.Collection, id string) *buptypes.CollectionChangeset {
	for _, changeset := range c.Changesets {
		if changeset.ID == id {
			return &changeset
		}
	}

	return nil
}

// TODO: put in types package?
type byPath []buptypes.File

func (s byPath) Len() int {
	return len(s)
}

func (s byPath) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPath) Less(i, j int) bool {
	return s[i].Path < s[j].Path
}
