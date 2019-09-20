package stoclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func clone(collectionId string, revisionId string, parentDir string, dirName string) error {
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

	return cloneCollection(filepath.Join(parentDir, dirName), revisionId, collection)
}

func cloneCollectionExistingDir(path string, revisionId string, collection *stotypes.Collection) error {
	// init this in "hack mode" (i.e. statefile not being read to memory). as soon as we
	// manage to write the statefile to disk, use normal procedure to init wd
	halfBakedWd := &workdirLocation{
		path: path,
	}

	manifestExists, err := fileexists.Exists(halfBakedWd.Join(localStatefile))
	if err != nil {
		return err
	}

	if manifestExists {
		return fmt.Errorf("%s already exists in %s - adopting would be dangerous", localStatefile, path)
	}

	if revisionId == "" {
		revisionId = collection.Head
	}

	halfBakedWd.manifest = &BupManifest{
		ChangesetId: revisionId,
		Collection:  *collection,
	}

	if err := halfBakedWd.SaveToDisk(); err != nil {
		return err
	}

	// now that properly initialized halfBakedWd was saved to disk (= bootstrapped),
	// reload it back from disk in a normal fashion
	wd, err := NewWorkdirLocation(halfBakedWd.path)
	if err != nil {
		return err
	}

	state, err := stateresolver.ComputeStateAt(*collection, wd.manifest.ChangesetId)
	if err != nil {
		return err
	}

	for _, file := range state.Files() {
		if err := cloneOneFile(wd, file); err != nil {
			return err
		}
	}

	return nil
}

// used both by collection create and collection download
func cloneCollection(path string, revisionId string, collection *stotypes.Collection) error {
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

	return cloneCollectionExistingDir(path, revisionId, collection)
}

func DownloadOneFile(file stotypes.File, destination io.Writer, config ClientConfig) error {
	for _, chunkDigest := range file.BlobRefs {
		blobRef, err := stotypes.BlobRefFromHex(chunkDigest)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)

		verifiedBody, closeBody, err := DownloadChunk(ctx, *blobRef, config)
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

func cloneOneFile(wd *workdirLocation, file stotypes.File) error {
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

	if err := DownloadOneFile(file, fileHandle, wd.clientConfig); err != nil {
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
		ezhttp.RespondsJson(collection, false))

	return collection, err
}

// verifies chunk integrity on-the-fly
func DownloadChunk(ctx context.Context, ref stotypes.BlobRef, clientConfig ClientConfig) (io.Reader, func(), error) {
	chunkDataRes, err := ezhttp.Get(
		ctx,
		clientConfig.UrlBuilder().DownloadBlob(ref.AsHex()),
		ezhttp.AuthBearer(clientConfig.AuthToken))
	if err != nil {
		return nil, func() {}, err
	}

	return stoutils.BlobHashVerifier(chunkDataRes.Body, ref), func() { chunkDataRes.Body.Close() }, nil
}
