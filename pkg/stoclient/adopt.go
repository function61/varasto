package stoclient

import (
	"context"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
)

func adopt(wd string, parentDirectoryId string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	clientConfig, err := ReadConfig()
	if err != nil {
		return err
	}

	// TODO: maybe the struct ctor should be codegen'd?
	collectionId, err := clientConfig.CommandClient().ExecExpectingCreatedRecordId(ctx, &stoservertypes.CollectionCreate{
		ParentDir: parentDirectoryId,
		Name:      filepath.Base(wd),
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
	return cloneCollectionExistingDir(wd, "", collection)
}

func adoptEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "adopt [parentDirectoryId]",
		Short: "Creates a new collection",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			wd, err := os.Getwd()
			panicIfError(err)

			panicIfError(adopt(wd, args[0]))
		},
	}
}
