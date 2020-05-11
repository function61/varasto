package stoserver

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) CollectionMoveFilesIntoAnotherCollection(cmd *stoservertypes.CollectionMoveFilesIntoAnotherCollection, ctx *command.Ctx) error {
	filesToMove := *cmd.Files

	if len(filesToMove) == 0 {
		return nil // no-op
	}

	if err := noDuplicates(filesToMove); err != nil {
		return err
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
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

		filesInSource := sourceState.Files()

		for _, filePathToMove := range filesToMove {
			fileToMove, found := filesInSource[filePathToMove]
			if !found {
				return fmt.Errorf("File to move not found: %s", filePathToMove)
			}

			createToDestination = append(createToDestination, fileToMove)
			deleteFromSource = append(deleteFromSource, fileToMove.Path)

			for _, refSerialized := range fileToMove.BlobRefs {
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
				if findDekEnvelope(blob.EncryptionKeyId, collDst.EncryptionKeys) == nil {
					dekEnvelope, err := copyAndReEncryptDekFromAnotherCollection(
						blob.EncryptionKeyId,
						extractKekPubKeyFingerprints(collSrc),
						tx,
						c.conf.KeyStore)
					if err != nil {
						return fmt.Errorf("copyAndReEncryptDekFromAnotherCollection: %v", err)
					}

					collDst.EncryptionKeys = append(collDst.EncryptionKeys, *dekEnvelope)
				}
			}
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

func (c *cHandlers) CollectionDeleteFiles(cmd *stoservertypes.CollectionDeleteFiles, ctx *command.Ctx) error {
	filesToDelete := *cmd.Files

	if len(filesToDelete) == 0 {
		return nil // no-op
	}

	if err := noDuplicates(filesToDelete); err != nil {
		return err
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Source)
		if err != nil {
			return err
		}

		stateForValidation, err := stateresolver.ComputeStateAt(*coll, coll.Head)
		if err != nil {
			return err
		}

		existingFiles := stateForValidation.Files()

		for _, fileToDelete := range filesToDelete {
			if _, has := existingFiles[fileToDelete]; !has {
				return fmt.Errorf("file to delete does not exist: %s", fileToDelete)
			}
		}

		changeset := stotypes.NewChangeset(
			stoutils.NewCollectionChangesetId(),
			coll.Head,
			ctx.Meta.Timestamp,
			nil,
			nil,
			filesToDelete)

		if err := appendAndValidateChangeset(changeset, coll); err != nil {
			return err
		}

		return stodb.CollectionRepository.Update(coll, tx)
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
	coll.BumpGlobalVersion()

	return nil
}

func minDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func noDuplicates(items []string) error {
	// once sorted, we can just assert that current item is not same as previous item
	sort.Strings(items)

	for i := 1; i < len(items); i++ {
		if items[i-1] == items[i] {
			return fmt.Errorf("duplicate item in list: %s", items[i])
		}
	}

	return nil
}
