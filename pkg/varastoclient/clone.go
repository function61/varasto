package varastoclient

import (
	"context"
	"errors"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func clone(collectionId string, revisionId string, parentDir string, dirName string) error {
	clientConfig, err := readConfig()
	if err != nil {
		return err
	}

	collection, err := fetchCollectionMetadata(*clientConfig, collectionId)
	if err != nil {
		return err
	}

	if dirName == "" {
		dirName = collection.Name
	}

	return cloneCollection(filepath.Join(parentDir, dirName), revisionId, collection)
}

func cloneCollectionExistingDir(path string, revisionId string, collection *varastotypes.Collection) error {
	// init this in "hack mode" (i.e. statefile not being read to memory). as soon as we
	// manage to write the statefile to disk, use normal procedure to init wd
	halfBakedWd := &workdirLocation{
		path: path,
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
func cloneCollection(path string, revisionId string, collection *varastotypes.Collection) error {
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

func cloneOneFile(wd *workdirLocation, file varastotypes.File) error {
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

	for _, chunkDigest := range file.BlobRefs {
		blobRef, err := varastotypes.BlobRefFromHex(chunkDigest)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
		defer cancel()

		chunkDataRes, err := ezhttp.Get(
			ctx,
			wd.clientConfig.ApiPath("/api/blobs/"+blobRef.AsHex()),
			ezhttp.AuthBearer(wd.clientConfig.AuthToken))
		if err != nil {
			return err
		}
		defer chunkDataRes.Body.Close()

		verifiedBody := varastoutils.BlobHashVerifier(chunkDataRes.Body, *blobRef)

		if _, err := io.Copy(fileHandle, verifiedBody); err != nil {
			return err
		}
	}

	fileHandle.Close() // even though we have the defer above - we probably need this for Chtimes()

	if err := os.Chtimes(filenameTemp, time.Now(), file.Modified); err != nil {
		return err
	}

	return os.Rename(filenameTemp, filename)
}

func fetchCollectionMetadata(clientConfig ClientConfig, id string) (*varastotypes.Collection, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	collection := &varastotypes.Collection{}
	_, err := ezhttp.Get(
		ctx,
		clientConfig.ApiPath("/api/collections/"+id),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.RespondsJson(collection, false))

	return collection, err
}
