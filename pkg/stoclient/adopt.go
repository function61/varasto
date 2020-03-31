package stoclient

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
)

func adopt(fullPath string, parentDirectoryId string) error {
	// cloneCollectionExistingDir() also checks for this, but we must do this before asking
	// server to create a collection because we don't want it to be created and error when
	// we begin cloning
	if err := assertStatefileNotExists(fullPath); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

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

	collection, err := FetchCollectionMetadata(*clientConfig, collectionId)
	if err != nil {
		return err
	}

	log.Printf("Collection %s created with id %s", collection.Name, collection.ID)

	// since we created an empty collection, there's actually nothing to download,
	// but this does other important housekeeping
	return cloneCollectionExistingDir(ctx, fullPath, "", collection)
}

func adoptEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "adopt [parentDirectoryId]",
		Short: "Adopts current directory as Varasto collection",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fullPath, err := os.Getwd()
			exitIfError(err)

			exitIfError(adopt(fullPath, args[0]))
		},
	}
}
