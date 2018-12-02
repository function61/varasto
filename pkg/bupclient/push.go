package bupclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/djherbis/times"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/bup/pkg/stateresolver"
	"github.com/function61/gokit/ezhttp"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const megabyte = 1024 * 1024

func computeChangeset(wd *workdirLocation) (*buptypes.CollectionChangeset, error) {
	parentState, err := stateresolver.ComputeStateAt(wd.manifest.Collection, wd.manifest.Collection.Head)
	if err != nil {
		return nil, err
	}

	// deleted during directory scan. what's left over is what are missing w.r.t. parent state
	filesMissing := parentState.Files()
	filesAtParent := parentState.Files() // this will not be mutated

	created := []buptypes.File{}
	updated := []buptypes.File{}

	errWalk := filepath.Walk(wd.path, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err // stop if encountering Walk() errors
		}

		if !fileInfo.IsDir() { // don't handle directories
			if fileInfo.Name() == localStatefile { // skip .bup files
				return nil
			}

			relativePath, errRel := filepath.Rel(wd.path, path)
			if errRel != nil {
				return errRel
			}

			// ok if key missing
			delete(filesMissing, relativePath)

			// https://unix.stackexchange.com/questions/2802/what-is-the-difference-between-modify-and-change-in-stat-command-context
			allTimes := times.Get(fileInfo)

			maybeCreationTime := fileInfo.ModTime()
			if allTimes.HasBirthTime() {
				maybeCreationTime = allTimes.BirthTime()
			}

			fil := buptypes.File{
				Path:     relativePath,
				Created:  maybeCreationTime,
				Modified: fileInfo.ModTime(),
				Size:     fileInfo.Size(),
				Sha256:   "",         // will be computed later
				BlobRefs: []string{}, // will be computed later
			}

			if before, existedBefore := filesAtParent[relativePath]; existedBefore {
				// TODO: this is really weak comparison
				if before.Size != fil.Size {
					updated = append(updated, fil)
				}
			} else {
				created = append(created, fil)
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

	return &buptypes.CollectionChangeset{
		ID:           buputils.NewCollectionChangesetId(),
		Parent:       wd.manifest.ChangesetId,
		Created:      time.Now(),
		FilesCreated: created,
		FilesUpdated: updated,
		FilesDeleted: deleted,
	}, nil
}

// returns ErrChunkMetadataNotFound if blob is not found
func blobExists(wd *workdirLocation, blobRef buptypes.BlobRef) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	// do a HEAD request to see if the blob exists
	resp, err := ezhttp.Send(
		ctx,
		http.MethodHead,
		wd.clientConfig.ApiPath("/blobs/"+blobRef.AsHex()),
		ezhttp.AuthBearer(wd.clientConfig.AuthToken))

	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, err // an actual error
	}

	return true, nil
}

func uploadChunks(wd *workdirLocation, bfile buptypes.File) (string, []string, error) {
	file, err := os.Open(wd.Join(bfile.Path))
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	pos := int64(0)
	chunkSize := int64(4 * megabyte)

	fullFileSha256 := sha256.New()
	chunkHashes := []string{}

	for {
		if _, err := file.Seek(pos, io.SeekStart); err != nil {
			return "", nil, err
		}

		chunk, errRead := ioutil.ReadAll(io.LimitReader(file, chunkSize))
		if errRead != nil {
			return "", nil, errRead
		}

		pos += chunkSize

		if len(chunk) == 0 {
			// should only happen if file size is exact multiple of chunkSize
			break
		}

		if _, err := fullFileSha256.Write(chunk); err != nil {
			return "", nil, err
		}

		fileSha256Bytes := sha256.Sum256(chunk)

		blobRef, err := buptypes.BlobRefFromHex(hex.EncodeToString(fileSha256Bytes[:]))
		if err != nil {
			return "", nil, err
		}

		chunkHashes = append(chunkHashes, blobRef.AsHex())

		// just check if the chunk exists already
		blobAlreadyExists, err := blobExists(wd, *blobRef)
		if err != nil {
			return "", nil, err
		}

		if blobAlreadyExists {
			log.Printf("Deduplicated chunk %s", blobRef.AsHex())
			continue
		}

		ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
		defer cancel()

		if _, err := ezhttp.Send(
			ctx,
			http.MethodPost,
			wd.clientConfig.ApiPath("/blobs/"+blobRef.AsHex()),
			ezhttp.AuthBearer(wd.clientConfig.AuthToken),
			ezhttp.SendBody(bytes.NewReader(chunk), "application/octet-stream")); err != nil {
			return "", nil, err
		}

		if int64(len(chunk)) < chunkSize {
			break
		}
	}

	return fmt.Sprintf("%x", fullFileSha256.Sum(nil)), chunkHashes, nil
}

func uploadChangeset(wd *workdirLocation, changeset buptypes.CollectionChangeset) (*buptypes.Collection, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	updatedCollection := &buptypes.Collection{}
	_, err := ezhttp.Send(
		ctx,
		http.MethodPost,
		wd.clientConfig.ApiPath("/collections/"+wd.manifest.Collection.ID+"/changesets"),
		ezhttp.AuthBearer(wd.clientConfig.AuthToken),
		ezhttp.SendJson(&changeset),
		ezhttp.RespondsJson(&updatedCollection, false))

	return updatedCollection, err
}

func push(wd *workdirLocation) error {
	ch, err := computeChangeset(wd)
	if err != nil {
		return err
	}

	if (len(ch.FilesCreated) + len(ch.FilesUpdated) + len(ch.FilesDeleted)) == 0 {
		log.Println("No files changed")
		return nil
	}

	// FIXME: correct this code duplication
	for i, created := range ch.FilesCreated {
		fileHash, chunkHashes, err := uploadChunks(wd, created)
		if err != nil {
			return err
		}

		created.Sha256 = fileHash
		created.BlobRefs = chunkHashes
		ch.FilesCreated[i] = created
	}

	for i, updated := range ch.FilesUpdated {
		fileHash, chunkHashes, err := uploadChunks(wd, updated)
		if err != nil {
			return err
		}

		updated.Sha256 = fileHash
		updated.BlobRefs = chunkHashes
		ch.FilesUpdated[i] = updated
	}

	updatedCollection, err := uploadChangeset(wd, *ch)
	if err != nil {
		return err
	}

	wd.manifest.ChangesetId = updatedCollection.Head
	wd.manifest.Collection = *updatedCollection

	return wd.SaveToDisk()
}
