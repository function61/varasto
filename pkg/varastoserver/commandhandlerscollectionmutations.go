package varastoserver

import (
	"errors"
	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"go.etcd.io/bbolt"
	"strings"
)

func (c *cHandlers) CollectionMoveFilesIntoAnotherCollection(cmd *CollectionMoveFilesIntoAnotherCollection, ctx *command.Ctx) error {
	// keep indexed map of filenames to move. they are removed on-the-fly, so in the end
	// we can check for len() == 0 to see that we saw them all
	filenamesToMove := map[string]bool{}
	for _, filename := range strings.Split(cmd.Files, ",") {
		filenamesToMove[filename] = true
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		collSrc, err := QueryWithTx(tx).Collection(cmd.Source)
		if err != nil {
			return err
		}

		collDst, err := QueryWithTx(tx).Collection(cmd.Destination)
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
			if _, shouldMove := filenamesToMove[file.Path]; !shouldMove {
				continue
			}

			deleteFromSource = append(deleteFromSource, file.Path)
			createToDestination = append(createToDestination, file)

			delete(filenamesToMove, file.Path)
		}

		if len(filenamesToMove) != 0 {
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

		collSrc.Changesets = append(collSrc.Changesets, srcChangeset)
		collSrc.Head = srcChangeset.ID

		collDst.Changesets = append(collDst.Changesets, dstChangeset)
		collDst.Head = dstChangeset.ID

		if err := CollectionRepository.Update(collSrc, tx); err != nil {
			return err
		}

		if err := CollectionRepository.Update(collDst, tx); err != nil {
			return err
		}

		return nil
	})
}
