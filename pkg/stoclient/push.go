package stoclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/djherbis/times"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	backgroundUploaderConcurrency = 3
)

func computeChangeset(wd *workdirLocation, bdl BlobDiscoveredListener) (*stotypes.CollectionChangeset, error) {
	parentState, err := stateresolver.ComputeStateAt(wd.manifest.Collection, wd.manifest.Collection.Head)
	if err != nil {
		return nil, err
	}

	collectionId := wd.manifest.Collection.ID

	// deleted during directory scan. what's left over is what are missing w.r.t. parent state
	filesMissing := parentState.Files()
	filesAtParent := parentState.Files() // this will not be mutated

	created := []stotypes.File{}
	updated := []stotypes.File{}

	errWalk := filepath.Walk(wd.path, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err // stop if encountering Walk() errors
		}

		if !fileInfo.IsDir() { // don't handle directories
			if fileInfo.Name() == localStatefile { // skip .varasto files
				return nil
			}

			// this returns \ on Windows, but we'll need to normalize to slashes for interop
			relativePath, errRel := filepath.Rel(wd.path, path)
			if errRel != nil {
				return errRel
			}
			relativePath = backslashesToForwardSlashes(relativePath)

			// ok if key missing
			delete(filesMissing, relativePath)

			if before, existedBefore := filesAtParent[relativePath]; existedBefore {
				definitelyChanged := before.Size != fileInfo.Size()
				maybeChanged := !before.Modified.Equal(fileInfo.ModTime())

				if definitelyChanged || maybeChanged {
					fil, err := scanFileAndDiscoverBlobs(wd.Join(relativePath), relativePath, fileInfo, collectionId, bdl)
					if err != nil {
						return err
					}

					// TODO: allow commit to change metadata like modification time or
					// execute bit?
					if before.Sha256 != fil.Sha256 {
						updated = append(updated, *fil)
					}
				}
			} else {
				fil, err := scanFileAndDiscoverBlobs(wd.Join(relativePath), relativePath, fileInfo, collectionId, bdl)
				if err != nil {
					return err
				}

				created = append(created, *fil)
			}
		}

		return nil
	})
	if errWalk != nil {
		return nil, errWalk
	}

	deleted := []string{}
	for missing, _ := range filesMissing {
		deleted = append(deleted, missing)
	}
	sort.Strings(deleted)

	ch := stotypes.NewChangeset(
		stoutils.NewCollectionChangesetId(),
		wd.manifest.ChangesetId,
		time.Now(),
		created,
		updated,
		deleted)

	return &ch, nil
}

// returns ErrChunkMetadataNotFound if blob is not found
func blobExists(blobRef stotypes.BlobRef, clientConfig ClientConfig) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	// do a HEAD request to see if the blob exists
	resp, err := ezhttp.Get(
		ctx,
		clientConfig.UrlBuilder().GetBlobMetadata(blobRef.AsHex()),
		ezhttp.AuthBearer(clientConfig.AuthToken))

	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, err // an actual error
	}

	return true, nil
}

func scanFileAndDiscoverBlobs(
	absolutePath string,
	relativePath string,
	fileInfo os.FileInfo,
	collectionId string,
	bdl BlobDiscoveredListener,
) (*stotypes.File, error) {
	// https://unix.stackexchange.com/questions/2802/what-is-the-difference-between-modify-and-change-in-stat-command-context
	allTimes := times.Get(fileInfo)

	maybeCreationTime := fileInfo.ModTime()
	if allTimes.HasBirthTime() {
		maybeCreationTime = allTimes.BirthTime()
	}

	bfile := &stotypes.File{
		Path:     relativePath,
		Created:  maybeCreationTime,
		Modified: fileInfo.ModTime(),
		Size:     fileInfo.Size(),
		Sha256:   "",         // will be computed later in this method
		BlobRefs: []string{}, // will be computed later in this method
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	pos := int64(0)

	fullContentHash := sha256.New()

	discoveryCancel := bdl.CancelCh()

	for {
		select {
		case <-discoveryCancel:
			return nil, errors.New("discovery canceled")
		default:
		}

		if _, err := file.Seek(pos, io.SeekStart); err != nil {
			return nil, err
		}

		chunk, errRead := ioutil.ReadAll(io.LimitReader(file, stotypes.BlobSize))
		if errRead != nil {
			return nil, errRead
		}

		pos += stotypes.BlobSize

		if len(chunk) == 0 {
			// should only happen if file size is exact multiple of blobSize
			break
		}

		if _, err := fullContentHash.Write(chunk); err != nil {
			return nil, err
		}

		chunkSha256Bytes := sha256.Sum256(chunk)

		blobRef, err := stotypes.BlobRefFromHex(hex.EncodeToString(chunkSha256Bytes[:]))
		if err != nil {
			return nil, err
		}

		bfile.BlobRefs = append(bfile.BlobRefs, blobRef.AsHex())

		bdl.BlobDiscovered(NewBlobDiscoveredAttrs(
			*blobRef,
			collectionId,
			chunk,
			stoutils.IsMaybeCompressible(relativePath),
			bfile.Path,
			bfile.Size))

		if int64(len(chunk)) < stotypes.BlobSize {
			break
		}
	}

	bfile.Sha256 = fmt.Sprintf("%x", fullContentHash.Sum(nil))

	return bfile, nil
}

func uploadChangeset(changeset stotypes.CollectionChangeset, collection stotypes.Collection, clientConfig ClientConfig) (*stotypes.Collection, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 60*3*time.Second)
	defer cancel()

	updatedCollection := &stotypes.Collection{}
	res, err := ezhttp.Post(
		ctx,
		clientConfig.UrlBuilder().CommitChangeset(collection.ID),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.SendJson(&changeset),
		ezhttp.RespondsJson(&updatedCollection, false))
	if err != nil {
		return nil, fmt.Errorf("error committing: %v", errSample(err, res))
	}

	return updatedCollection, nil
}

func pushOne(collectionId string, path string) error {
	clientConfig, err := ReadConfig()
	if err != nil {
		return err
	}

	coll, err := FetchCollectionMetadata(*clientConfig, collectionId)
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	absolutePath := filepath.Join(wd, path)

	fileInfo, err := os.Stat(absolutePath)
	if err != nil {
		return err
	}

	buploader := NewBackgroundUploader(
		backgroundUploaderConcurrency,
		*clientConfig,
		textUiUploadProgressOutputIfInTerminal())

	file, err := scanFileAndDiscoverBlobs(absolutePath, path, fileInfo, collectionId, buploader)
	if err != nil {
		return err
	}

	if err := buploader.WaitFinished(); err != nil {
		return err
	}

	changeset := stotypes.NewChangeset(
		stoutils.NewCollectionChangesetId(),
		coll.Head,
		time.Now(),
		[]stotypes.File{*file},
		[]stotypes.File{},
		[]string{})

	_, err = uploadChangeset(changeset, *coll, *clientConfig)
	return err
}

func push(wd *workdirLocation) error {
	buploader := NewBackgroundUploader(
		backgroundUploaderConcurrency,
		wd.clientConfig,
		textUiUploadProgressOutputIfInTerminal())

	ch, err := computeChangeset(wd, buploader)
	if err != nil {
		return err
	}

	if !ch.AnyChanges() {
		log.Println("No files changed")
		return nil
	}

	if err := buploader.WaitFinished(); err != nil {
		return err
	}

	updatedCollection, err := uploadChangeset(*ch, wd.manifest.Collection, wd.clientConfig)
	if err != nil {
		return err
	}

	wd.manifest.ChangesetId = updatedCollection.Head
	wd.manifest.Collection = *updatedCollection

	return wd.SaveToDisk()
}

func backslashesToForwardSlashes(in string) string {
	return strings.Replace(in, `\`, "/", -1)
}

func errSample(err error, response *http.Response) error {
	sample := []byte("(no response)")
	if response != nil {
		sample, _ = ioutil.ReadAll(io.LimitReader(response.Body, 256))
	}

	return fmt.Errorf("%v: %s", err, sample)
}
