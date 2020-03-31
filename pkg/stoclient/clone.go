package stoclient

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
)

func clone(
	ctx context.Context,
	collectionId string,
	revisionId string,
	parentDir string,
	dirName string,
) error {
	clientConfig, err := ReadConfig()
	if err != nil {
		return err
	}

	collection, err := FetchCollectionMetadata(*clientConfig, collectionId)
	if err != nil {
		return err
	}

	if dirName == "" {
		dirName = collection.Name
	}

	return cloneCollection(ctx, filepath.Join(parentDir, dirName), revisionId, collection)
}

func cloneCollectionExistingDir(
	ctx context.Context,
	path string,
	revisionId string,
	collection *stotypes.Collection,
) error {
	if err := assertStatefileNotExists(path); err != nil {
		return err
	}

	if revisionId == "" {
		revisionId = collection.Head
	}

	if err := (&workdirLocation{
		path: path,
		manifest: &BupManifest{
			ChangesetId: revisionId,
			Collection:  *collection,
		},
	}).SaveToDisk(); err != nil {
		return err
	}

	// now that properly initialized manifest was saved to disk (= bootstrapped),
	// reload it back from disk in a normal fashion
	wd, err := NewWorkdirLocation(path)
	if err != nil {
		return err
	}

	state, err := stateresolver.ComputeStateAt(*collection, wd.manifest.ChangesetId)
	if err != nil {
		return err
	}

	for _, file := range state.Files() {
		if err := cloneOneFile(ctx, wd, file); err != nil {
			return err
		}
	}

	return nil
}

// used both by collection create and collection download
func cloneCollection(
	ctx context.Context,
	path string,
	revisionId string,
	collection *stotypes.Collection,
) error {
	// init this in "hack mode" (i.e. statefile not being read to memory). as soon as we
	// manage to write the statefile to disk, use normal procedure to init wd
	halfBakedWd := &workdirLocation{
		path: path,
	}

	dirAlreadyExists, err := fileexists.Exists(halfBakedWd.Join("/"))
	if err != nil {
		return err
	}

	if dirAlreadyExists {
		return errors.New("dir-to-clone-to already exists!")
	}

	if err := os.Mkdir(halfBakedWd.Join("/"), 0700); err != nil {
		return err
	}

	return cloneCollectionExistingDir(ctx, path, revisionId, collection)
}

func DownloadOneFile(
	ctx context.Context,
	file stotypes.File,
	collectionId string,
	destination io.Writer,
	config ClientConfig,
) error {
	for _, chunkDigest := range file.BlobRefs {
		blobRef, err := stotypes.BlobRefFromHex(chunkDigest)
		if err != nil {
			return err
		}

		childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)

		verifiedBody, closeBody, err := DownloadChunk(childCtx, *blobRef, collectionId, config)
		if err != nil {
			cancel()
			return err
		}

		if _, err := io.Copy(destination, verifiedBody); err != nil {
			cancel()
			closeBody()
			return err
		}

		cancel()

		closeBody()
	}

	return nil
}

func cloneOneFile(ctx context.Context, wd *workdirLocation, file stotypes.File) error {
	log.Printf("Downloading %s", file.Path)

	filename := wd.Join(file.Path)
	filenameTemp := filename + ".temp"

	// does not error if already exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	fileHandle, err := os.Create(filenameTemp)
	if err != nil {
		return err
	}
	defer fileHandle.Close()

	if err := DownloadOneFile(ctx, file, wd.manifest.Collection.ID, fileHandle, wd.clientConfig); err != nil {
		return err
	}

	fileHandle.Close() // even though we have the defer above - we probably need this for Chtimes()

	if err := os.Chtimes(filenameTemp, time.Now(), file.Modified); err != nil {
		return err
	}

	return os.Rename(filenameTemp, filename)
}

func FetchCollectionMetadata(clientConfig ClientConfig, id string) (*stotypes.Collection, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	collection := &stotypes.Collection{}
	_, err := ezhttp.Get(
		ctx,
		clientConfig.UrlBuilder().GetCollection(id),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.RespondsJson(collection, false),
		ezhttp.Client(clientConfig.HttpClient()))

	return collection, err
}

// verifies chunk integrity on-the-fly
func DownloadChunk(ctx context.Context, ref stotypes.BlobRef, collectionId string, clientConfig ClientConfig) (io.Reader, func(), error) {
	chunkDataRes, err := ezhttp.Get(
		ctx,
		clientConfig.UrlBuilder().DownloadBlob(ref.AsHex(), collectionId),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.Client(clientConfig.HttpClient()))
	if err != nil {
		return nil, func() {}, err
	}

	return stoutils.BlobHashVerifier(chunkDataRes.Body, ref), func() { chunkDataRes.Body.Close() }, nil
}
