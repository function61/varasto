package stoserver

import (
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
	"strings"
	"time"
)

func (c *cHandlers) CollectionMoveFilesIntoAnotherCollection(cmd *stoservertypes.CollectionMoveFilesIntoAnotherCollection, ctx *command.Ctx) error {
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

		sourceState, err := stateresolver.ComputeStateAt(*collSrc, collSrc.Head)
		if err != nil {
			return err
		}

		deleteFromSource := []string{}
		createToDestination := []stotypes.File{}

		for _, file := range sourceState.Files() {
			if _, shouldMove := hashesToMove[file.Sha256]; !shouldMove {
				continue
			}

			delete(hashesToMove, file.Sha256)

			deleteFromSource = append(deleteFromSource, file.Path)
			createToDestination = append(createToDestination, file)

			for _, refSerialized := range file.BlobRefs {
				ref, err := stotypes.BlobRefFromHex(refSerialized)
				if err != nil {
					return err
				}

				blob, err := stodb.Read(tx).Blob(*ref)
				if err != nil {
					return err
				}

				// if destination collection doesn't have encryption key for this blob,
				// copy it over
				if stotypes.FindKeyById(blob.EncryptionKeyId, collDst.EncryptionKeys) == nil {
					keyToCopy := stotypes.FindKeyById(blob.EncryptionKeyId, collSrc.EncryptionKeys)
					if keyToCopy == nil {
						return fmt.Errorf(
							"cannot find key envelope %s from source collection",
							blob.EncryptionKeyId)
					}

					collDst.EncryptionKeys = append(collDst.EncryptionKeys, *keyToCopy)
				}
			}
		}

		if len(hashesToMove) != 0 {
			return errors.New("did not find all files to move")
		}

		srcChangeset := stotypes.NewChangeset(
			stoutils.NewCollectionChangesetId(),
			collSrc.Head,
			ctx.Meta.Timestamp,
			nil,
			nil,
			deleteFromSource)

		// duplicate filenames are asserted by appendAndValidateChangeset()
		dstChangeset := stotypes.NewChangeset(
			stoutils.NewCollectionChangesetId(),
			collDst.Head,
			ctx.Meta.Timestamp,
			createToDestination,
			nil,
			nil)

		if err := appendAndValidateChangeset(srcChangeset, collSrc); err != nil {
			return err
		}

		if err := appendAndValidateChangeset(dstChangeset, collDst); err != nil {
			return err
		}

		if err := stodb.CollectionRepository.Update(collSrc, tx); err != nil {
			return err
		}

		if err := stodb.CollectionRepository.Update(collDst, tx); err != nil {
			return err
		}

		return nil
	})
}

func appendAndValidateChangeset(changeset stotypes.CollectionChangeset, coll *stotypes.Collection) error {
	currentHeadState, err := stateresolver.ComputeStateAt(*coll, coll.Head)
	if err != nil {
		return err
	}
	filesAtCurrentHead := currentHeadState.Files()

	for _, file := range changeset.FilesCreated {
		coll.Created = minDate(coll.Created, file.Created)
		coll.Created = minDate(coll.Created, file.Modified)

		if _, exists := filesAtCurrentHead[file.Path]; exists {
			return fmt.Errorf("cannot create file %s because it already exists at revision %s", file.Path, coll.Head)
		}
	}

	for _, file := range changeset.FilesUpdated {
		coll.Created = minDate(coll.Created, file.Created)
		coll.Created = minDate(coll.Created, file.Modified)

		if _, exists := filesAtCurrentHead[file.Path]; !exists {
			return fmt.Errorf("cannot update file %s because it does not exist at revision %s", file.Path, coll.Head)
		}
	}

	for _, file := range changeset.FilesDeleted {
		if _, exists := filesAtCurrentHead[file]; !exists {
			return fmt.Errorf("cannot delete file %s because it does not exist at revision %s", file, coll.Head)
		}
	}

	coll.Changesets = append(coll.Changesets, changeset)
	coll.Head = changeset.ID

	return nil
}

func minDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
