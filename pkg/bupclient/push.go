package bupclient

import (
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

const (
	megabyte  = 1024 * 1024
	chunkSize = 4 * megabyte
)

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

			if before, existedBefore := filesAtParent[relativePath]; existedBefore {
				definitelyChanged := before.Size != fileInfo.Size()
				maybeChanged := !before.Modified.Equal(fileInfo.ModTime())

				if definitelyChanged || maybeChanged {
					fil, err := analyzeFileForChanges(wd, relativePath, fileInfo)
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
				fil, err := analyzeFileForChanges(wd, relativePath, fileInfo)
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

func analyzeFileForChanges(wd *workdirLocation, relativePath string, fileInfo os.FileInfo) (*buptypes.File, error) {
	// https://unix.stackexchange.com/questions/2802/what-is-the-difference-between-modify-and-change-in-stat-command-context
	allTimes := times.Get(fileInfo)

	maybeCreationTime := fileInfo.ModTime()
	if allTimes.HasBirthTime() {
		maybeCreationTime = allTimes.BirthTime()
	}

	bfile := &buptypes.File{
		Path:     relativePath,
		Created:  maybeCreationTime,
		Modified: fileInfo.ModTime(),
		Size:     fileInfo.Size(),
		Sha256:   "",         // will be computed later in this method
		BlobRefs: []string{}, // will be computed later in this method
	}

	file, err := os.Open(wd.Join(bfile.Path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	pos := int64(0)

	fullContentHash := sha256.New()

	for {
		if _, err := file.Seek(pos, io.SeekStart); err != nil {
			return nil, err
		}

		chunk, errRead := ioutil.ReadAll(io.LimitReader(file, chunkSize))
		if errRead != nil {
			return nil, errRead
		}

		pos += chunkSize

		if len(chunk) == 0 {
			// should only happen if file size is exact multiple of chunkSize
			break
		}

		if _, err := fullContentHash.Write(chunk); err != nil {
			return nil, err
		}

		fileSha256Bytes := sha256.Sum256(chunk)

		blobRef, err := buptypes.BlobRefFromHex(hex.EncodeToString(fileSha256Bytes[:]))
		if err != nil {
			return nil, err
		}

		bfile.BlobRefs = append(bfile.BlobRefs, blobRef.AsHex())

		if int64(len(chunk)) < chunkSize {
			break
		}
	}

	bfile.Sha256 = fmt.Sprintf("%x", fullContentHash.Sum(nil))

	return bfile, nil
}

func uploadChunks(wd *workdirLocation, bfile buptypes.File) error {
	file, err := os.Open(wd.Join(bfile.Path))
	if err != nil {
		return err
	}
	defer file.Close()

	for blobIdx, brHex := range bfile.BlobRefs {
		blobRef, err := buptypes.BlobRefFromHex(brHex)
		if err != nil {
			return err
		}

		// just check if the chunk exists already
		blobAlreadyExists, err := blobExists(wd, *blobRef)
		if err != nil {
			return err
		}

		if blobAlreadyExists {
			log.Printf("Deduplicated chunk %s", blobRef.AsHex())
			continue
		}

		if _, err := file.Seek(int64(blobIdx*chunkSize), io.SeekStart); err != nil {
			return err
		}

		chunk := io.LimitReader(file, chunkSize)

		ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
		defer cancel()

		if _, err := ezhttp.Send(
			ctx,
			http.MethodPost,
			wd.clientConfig.ApiPath("/blobs/"+blobRef.AsHex()),
			ezhttp.AuthBearer(wd.clientConfig.AuthToken),
			ezhttp.SendBody(buputils.BlobHashVerifier(chunk, *blobRef), "application/octet-stream")); err != nil {
			return err
		}
	}

	return nil
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

	for _, created := range ch.FilesCreated {
		if err := uploadChunks(wd, created); err != nil {
			return err
		}
	}

	for _, updated := range ch.FilesUpdated {
		if err := uploadChunks(wd, updated); err != nil {
			return err
		}
	}

	updatedCollection, err := uploadChangeset(wd, *ch)
	if err != nil {
		return err
	}

	wd.manifest.ChangesetId = updatedCollection.Head
	wd.manifest.Collection = *updatedCollection

	return wd.SaveToDisk()
}
