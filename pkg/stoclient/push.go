package stoclient

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/minio/sha256-simd"
)

const (
	BackgroundUploaderConcurrency = 3
)

func computeChangeset(ctx context.Context, wd *workdirLocation, bdl BlobDiscoveredListener) (*stotypes.CollectionChangeset, error) {
	parentState, err := stateresolver.ComputeStateAtHead(wd.manifest.Collection)
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
					fil, err := scanFileAndDiscoverBlobs(ctx, wd.Join(relativePath), relativePath, fileInfo, collectionId, bdl)
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
				fil, err := scanFileAndDiscoverBlobs(ctx, wd.Join(relativePath), relativePath, fileInfo, collectionId, bdl)
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
	for missing := range filesMissing {
		deleted = append(deleted, missing)
	}
	sort.Strings(deleted)

	ch := stotypes.NewChangeset(
		stoutils.NewCollectionChangesetId(),
		wd.manifest.ChangesetId, // parent
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
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.Client(clientConfig.HttpClient()))

	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, err // an actual error
	}

	return true, nil
}

func scanFileAndDiscoverBlobs(
	ctx context.Context,
	absolutePath string,
	relativePath string,
	fileInfo os.FileInfo,
	collectionId string,
	bdl BlobDiscoveredListener,
) (*stotypes.File, error) {
	file, err := os.Open(absolutePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// https://unix.stackexchange.com/questions/2802/what-is-the-difference-between-modify-and-change-in-stat-command-context
	allTimes := times.Get(fileInfo)

	maybeCreationTime := fileInfo.ModTime()
	if allTimes.HasBirthTime() {
		maybeCreationTime = allTimes.BirthTime()
	}

	return ScanAndDiscoverBlobs(
		ctx,
		relativePath,
		file,
		fileInfo.Size(),
		maybeCreationTime,
		fileInfo.ModTime(),
		collectionId,
		bdl)
}

// - for when you don't have a file, but you have a stream
// - totalsize is used for progress calculation, but if you're not using progress UI you
// can set it to 0
func ScanAndDiscoverBlobs(
	ctx context.Context,
	relativePath string,
	file io.Reader,
	totalSize int64,
	creationTime time.Time,
	modifiedTime time.Time,
	collectionId string,
	bdl BlobDiscoveredListener,
) (*stotypes.File, error) {
	bfile := &stotypes.File{
		Path:     relativePath,
		Created:  creationTime,
		Modified: modifiedTime,
		Size:     0,          // computed later
		Sha256:   "",         // computed later
		BlobRefs: []string{}, // computed later
	}

	fullContentHash := sha256.New()

	discoveryCancel := bdl.CancelCh()

	for {
		select {
		case <-discoveryCancel:
			return nil, errors.New("discovery canceled")
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		chunk, errRead := ioutil.ReadAll(io.LimitReader(file, stotypes.BlobSize))
		if errRead != nil {
			return nil, errRead
		}

		if len(chunk) == 0 {
			// should only happen if file size is exact multiple of blobSize
			break
		}

		bfile.Size += int64(len(chunk))

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
			totalSize))
	}

	bfile.Sha256 = fmt.Sprintf("%x", fullContentHash.Sum(nil))

	return bfile, nil
}

func Commit(
	changeset stotypes.CollectionChangeset,
	collectionId string,
	clientConfig ClientConfig,
) (*stotypes.Collection, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 60*3*time.Second)
	defer cancel()

	updatedCollection := &stotypes.Collection{}
	if _, err := ezhttp.Post(
		ctx,
		clientConfig.UrlBuilder().CommitChangeset(collectionId),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.SendJson(&changeset),
		ezhttp.RespondsJson(&updatedCollection, false),
		ezhttp.Client(clientConfig.HttpClient()),
	); err != nil {
		return nil, fmt.Errorf("CommitChangeset: %v", err)
	}

	return updatedCollection, nil
}

func pushOne(ctx context.Context, collectionId string, path string) error {
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
		ctx,
		BackgroundUploaderConcurrency,
		*clientConfig,
		textUiUploadProgressOutputIfInTerminal())

	file, err := scanFileAndDiscoverBlobs(ctx, absolutePath, path, fileInfo, collectionId, buploader)
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

	_, err = Commit(changeset, coll.ID, *clientConfig)
	return err
}

func push(ctx context.Context, wd *workdirLocation) error {
	buploader := NewBackgroundUploader(
		ctx,
		BackgroundUploaderConcurrency,
		wd.clientConfig,
		textUiUploadProgressOutputIfInTerminal())

	ch, err := computeChangeset(ctx, wd, buploader)
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

	updatedCollection, err := Commit(*ch, wd.manifest.Collection.ID, wd.clientConfig)
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
