package varastoserver

import (
	"errors"
	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/varastoserver/stodb"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"go.etcd.io/bbolt"
	"strings"
	"time"
)

func (c *cHandlers) CollectionMoveFilesIntoAnotherCollection(cmd *CollectionMoveFilesIntoAnotherCollection, ctx *command.Ctx) error {
	if true {
		return errors.New("cannot use before changing blobs' owners and taking encryption keys into account")
	}

	// keep indexed map of filenames to move. they are removed on-the-fly, so in the end
	// we can check for len() == 0 to see that we saw them all
	hashesToMove := map[string]bool{}
	for _, hash := range strings.Split(cmd.Files, ",") {
		hashesToMove[hash] = true
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		collSrc, err := stodb.Read(tx).Collection(cmd.Source)
		if err != nil {
			return err
		}

		collDst, err := stodb.Read(tx).Collection(cmd.Destination)
		if err != nil {
			return err
		}

		if cmd.Source == cmd.Destination {
			return errors.New("Source and destination cannot be same")
		}

		state, err := stateresolver.ComputeStateAt(*collSrc, collSrc.Head)
		if err != nil {
			return err
		}

		deleteFromSource := []string{}
		createToDestination := []varastotypes.File{}

		for _, file := range state.Files() {
			if _, shouldMove := hashesToMove[file.Sha256]; shouldMove {
				delete(hashesToMove, file.Sha256)
			} else {
				continue
			}

			deleteFromSource = append(deleteFromSource, file.Path)
			createToDestination = append(createToDestination, file)
		}

		if len(hashesToMove) != 0 {
			return errors.New("did not find all files to move")
		}

		srcChangeset := varastotypes.NewChangeset(
			varastoutils.NewCollectionChangesetId(),
			collSrc.Head,
			ctx.Meta.Timestamp,
			nil,
			nil,
			deleteFromSource)

		dstChangeset := varastotypes.NewChangeset(
			varastoutils.NewCollectionChangesetId(),
			collDst.Head,
			ctx.Meta.Timestamp,
			createToDestination,
			nil,
			nil)

		appendChangeset(srcChangeset, collSrc)
		appendChangeset(dstChangeset, collDst)

		if err := stodb.CollectionRepository.Update(collSrc, tx); err != nil {
			return err
		}

		if err := stodb.CollectionRepository.Update(collDst, tx); err != nil {
			return err
		}

		return nil
	})
}

func appendChangeset(changeset varastotypes.CollectionChangeset, coll *varastotypes.Collection) {
	for _, file := range changeset.FilesCreated {
		coll.Created = minDate(coll.Created, file.Created)
		coll.Created = minDate(coll.Created, file.Modified)
	}

	for _, file := range changeset.FilesUpdated {
		coll.Created = minDate(coll.Created, file.Created)
		coll.Created = minDate(coll.Created, file.Modified)
	}

	coll.Changesets = append(coll.Changesets, changeset)
	coll.Head = changeset.ID
}

func minDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
