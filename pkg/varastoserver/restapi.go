package varastoserver

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/varastoserver/varastointegrityverifier"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func defineRestApi(
	router *mux.Router,
	conf *ServerConfig,
	db *bolt.DB,
	ivController *varastointegrityverifier.Controller,
	mwares httpauth.MiddlewareChainMap,
	logger *log.Logger,
) error {
	var han HttpHandlers = &handlers{db, conf, ivController}

	// v2 endpoints
	RegisterRoutes(han, mwares, muxregistrator.New(router))

	// legacy (TODO: move these to v2)
	return defineLegacyRestApi(router, conf, db, logger)
}

func defineLegacyRestApi(router *mux.Router, conf *ServerConfig, db *bolt.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	getCollection := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		coll, err := QueryWithTx(tx).Collection(mux.Vars(r)["collectionId"])
		if err != nil {
			if err == blorm.ErrNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		outJson(w, coll)
	}

	newCollection := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		tx, err := db.Begin(true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		req := &varastotypes.CreateCollectionRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		collection, err := saveNewCollection(req.ParentDirectoryId, req.Name, tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		panicIfError(tx.Commit())

		outJson(w, collection)
	}

	uploadBlob := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		// we need a hint from the client of what the collection is, so we can resolve a
		// volume onto which the blob should be stored
		collectionId := r.URL.Query().Get("collection")

		blobRef, err := varastotypes.BlobRefFromHex(mux.Vars(r)["blobRef"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tx, errTxBegin := db.Begin(true)
		panicIfError(errTxBegin)
		defer tx.Rollback()

		if _, err := QueryWithTx(tx).Blob(*blobRef); err != blorm.ErrNotFound {
			http.Error(w, "blob already exists!", http.StatusBadRequest)
			return
		}

		coll, err := QueryWithTx(tx).Collection(collectionId)
		panicIfError(err)

		volumeId := coll.DesiredVolumes[0]

		volumeDriver, driverFound := conf.VolumeDrivers[volumeId]
		if !driverFound {
			http.Error(w, "volume driver not found", http.StatusInternalServerError)
			return
		}

		blobSizeBytes, err := volumeDriver.Store(*blobRef, varastoutils.BlobHashVerifier(r.Body, *blobRef))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		panicIfError(err)

		logl.Debug.Printf("wrote blob %s", blobRef.AsHex())

		panicIfError(volumeManagerIncreaseBlobCount(tx, volumeId, blobSizeBytes))

		panicIfError(BlobRepository.Update(&varastotypes.Blob{
			Ref:        *blobRef,
			Volumes:    []int{volumeId},
			Referenced: false,
		}, tx))

		panicIfError(tx.Commit())
	}

	commitChangeset := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		collectionId := mux.Vars(r)["collectionId"]

		var changeset varastotypes.CollectionChangeset
		panicIfError(json.NewDecoder(r.Body).Decode(&changeset))

		tx, errTxBegin := db.Begin(true)
		panicIfError(errTxBegin)
		defer tx.Rollback()

		coll, err := QueryWithTx(tx).Collection(collectionId)
		panicIfError(err)

		if collectionHasChangesetId(changeset.ID, coll) {
			http.Error(w, "changeset ID already in collection", http.StatusBadRequest)
			return
		}

		if changeset.Parent != varastotypes.NoParentId && !collectionHasChangesetId(changeset.Parent, coll) {
			http.Error(w, "parent changeset not found", http.StatusBadRequest)
			return
		}

		if changeset.Parent != coll.Head {
			// TODO: force push or rebase support?
			http.Error(w, "commit does not target current head. would result in dangling heads!", http.StatusBadRequest)
			return
		}

		createdAndUpdated := append(changeset.FilesCreated, changeset.FilesUpdated...)

		for _, file := range createdAndUpdated {
			for _, refHex := range file.BlobRefs {
				ref, err := varastotypes.BlobRefFromHex(refHex)
				if err != nil {
					panic(err)
				}

				blob, err := QueryWithTx(tx).Blob(*ref)
				if err != nil {
					http.Error(w, fmt.Sprintf("blob %s not found", ref.AsHex()), http.StatusBadRequest)
					return
				}

				// FIXME: if same changeset mentions same blob many times, we update the old blob
				// metadata many times due to the transaction reads not seeing uncommitted writes
				blob.Referenced = true
				blob.VolumesPendingReplication = missingFromLeftHandSide(
					blob.Volumes,
					coll.DesiredVolumes)
				blob.IsPendingReplication = len(blob.VolumesPendingReplication) > 0

				panicIfError(BlobRepository.Update(blob, tx))
			}
		}

		// update head pointer & calc Created timestamp
		appendChangeset(changeset, coll)

		panicIfError(CollectionRepository.Update(coll, tx))
		panicIfError(tx.Commit())

		logl.Info.Printf("Collection %s changeset %s committed", coll.ID, changeset.ID)

		outJson(w, coll)
	}

	// shared by getBlob(), getBlobHead()
	getBlobCommon := func(blobRefSerialized string, w http.ResponseWriter) (*varastotypes.BlobRef, *varastotypes.Blob) {
		blobRef, err := varastotypes.BlobRefFromHex(blobRefSerialized)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, nil
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		blobMetadata, err := QueryWithTx(tx).Blob(*blobRef)
		if err != nil {
			if err == blorm.ErrNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
				return nil, nil
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil, nil
			}
		}

		return blobRef, blobMetadata
	}

	// returns 404 if blob not found
	getBlobHead := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		_, blobMetadata := getBlobCommon(mux.Vars(r)["blobRef"], w)
		if blobMetadata == nil {
			return // error was handled in common method
		}

		// don't return anything else
	}

	// returns 404 if blob not found
	getBlob := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		blobRef, blobMetadata := getBlobCommon(mux.Vars(r)["blobRef"], w)
		if blobMetadata == nil {
			return // error was handled in common method
		}

		// try to find the first local volume that has this blob
		var foundDriver blobdriver.Driver
		for _, volumeId := range blobMetadata.Volumes {
			if driver, found := conf.VolumeDrivers[volumeId]; found {
				foundDriver = driver
				break
			}
		}

		// TODO: issue HTTP redirect to correct node?
		if foundDriver == nil {
			http.Error(w, varastotypes.ErrBlobNotAccessibleOnThisNode.Error(), http.StatusInternalServerError)
			return
		}

		file, err := foundDriver.Fetch(*blobRef)
		if err != nil {
			if os.IsNotExist(err) {
				// should not happen, because metadata said that we should have this blob
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if _, err := io.Copy(w, varastoutils.BlobHashVerifier(file, *blobRef)); err != nil {
			// FIXME: shouldn't try to write headers if even one write went to ResponseWriter
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	router.HandleFunc("/api/blobs/{blobRef}", getBlob).Methods(http.MethodGet)
	router.HandleFunc("/api/blobs/{blobRef}", getBlobHead).Methods(http.MethodHead)
	router.HandleFunc("/api/blobs/{blobRef}", uploadBlob).Methods(http.MethodPost)

	router.HandleFunc("/api/collections", newCollection).Methods(http.MethodPost)
	router.HandleFunc("/api/collections/{collectionId}", getCollection).Methods(http.MethodGet)
	router.HandleFunc("/api/collections/{collectionId}/changesets", commitChangeset).Methods(http.MethodPost)

	return nil
}

func saveNewCollection(parentDirectoryId string, name string, tx *bolt.Tx) (*varastotypes.Collection, error) {
	if _, err := QueryWithTx(tx).Directory(parentDirectoryId); err != nil {
		if err == blorm.ErrNotFound {
			return nil, errors.New("parent directory not found")
		} else {
			return nil, err
		}
	}

	// TODO: resolve this from closest parent that has policy defined?
	replicationPolicy, err := QueryWithTx(tx).ReplicationPolicy("default")
	if err != nil {
		return nil, err
	}

	encryptionKey := [32]byte{}
	if _, err := rand.Read(encryptionKey[:]); err != nil {
		return nil, err
	}

	collection := &varastotypes.Collection{
		ID:             varastoutils.NewCollectionId(),
		Created:        time.Now(),
		Directory:      parentDirectoryId,
		Name:           name,
		DesiredVolumes: replicationPolicy.DesiredVolumes,
		Head:           varastotypes.NoParentId,
		EncryptionKey:  encryptionKey,
		Changesets:     []varastotypes.CollectionChangeset{},
		Metadata:       map[string]string{},
	}

	// highly unlikely
	if _, err := QueryWithTx(tx).Collection(collection.ID); err != blorm.ErrNotFound {
		return nil, errors.New("accidentally generated duplicate collection ID")
	}

	return collection, CollectionRepository.Update(collection, tx)
}
