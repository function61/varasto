package bupclient

import (
	"context"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/ezhttp"
	"log"
	"path/filepath"
)

func mk(parentPath string, parentDirectoryId string, collectionName string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	clientConfig, err := readConfig()
	if err != nil {
		return err
	}

	collection := buptypes.Collection{}

	_, err = ezhttp.Post(
		ctx,
		clientConfig.ApiPath("/api/collections"),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.SendJson(&buptypes.CreateCollectionRequest{
			Name:              collectionName,
			ParentDirectoryId: parentDirectoryId,
		}),
		ezhttp.RespondsJson(&collection, false))

	if err != nil {
		return err
	}

	log.Printf("Collection %s created with id %s", collection.Name, collection.ID)

	// since we created an empty collection, there's actually nothing to download,
	// but this does other important housekeeping
	return cloneCollection(filepath.Join(parentPath, collection.Name), "", &collection)
}
