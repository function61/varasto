package stoclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
)

func adopt(ctx context.Context, fullPath string, parentDirectoryId string) error {
	// cloneCollectionExistingDir() also checks for this, but we must do this before asking
	// server to create a collection because we don't want it to be created and error when
	// we begin cloning
	if err := assertStatefileNotExists(fullPath); err != nil {
		return err
	}

	clientConfig, err := ReadConfig()
	if err != nil {
		return err
	}

	// TODO: maybe the struct ctor should be codegen'd?
	collectionId, err := clientConfig.CommandClient().ExecExpectingCreatedRecordId(ctx, &stoservertypes.CollectionCreate{
		ParentDir: parentDirectoryId,
		Name:      filepath.Base(fullPath),
	})
	if err != nil {
		return err
	}

	collection, err := clientConfig.Client().FetchCollectionMetadata(ctx, collectionId)
	if err != nil {
		return err
	}

	log.Printf("Collection %s created with id %s", collection.Name, collection.ID)

	// since we created an empty collection, there's actually nothing to download,
	// but this does other important housekeeping
	return cloneCollectionExistingDir(ctx, fullPath, "", collection)
}

func adoptEntrypoint() *cobra.Command {
	push := false

	cmd := &cobra.Command{
		Use:   "adopt [parentDirectoryId]",
		Short: "Adopts current directory as Varasto collection",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(func(parentDirectoryId string) error {
				ctx := osutil.CancelOnInterruptOrTerminate(nil)

				fullPath, err := os.Getwd()
				if err != nil {
					return err
				}

				if err := adopt(ctx, fullPath, parentDirectoryId); err != nil {
					return err
				}

				if push {
					if err := pushCurrentWorkdir(ctx); err != nil {
						return fmt.Errorf("push: %w", err)
					}
				}

				return nil
			}(args[0]))
		},
	}

	cmd.Flags().BoolVarP(&push, "push", "", push, "Push after adopting")

	return cmd
}
