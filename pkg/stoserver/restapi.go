package stoserver

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
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
	ivController *stointegrityverifier.Controller,
	mwares httpauth.MiddlewareChainMap,
	logger *log.Logger,
) error {
	var han stoservertypes.HttpHandlers = &handlers{db, conf, ivController, logger}

	// v2 endpoints
	stoservertypes.RegisterRoutes(han, mwares, muxregistrator.New(router))

	// legacy (TODO: move these to v2)
	return defineLegacyRestApi(router, conf, db)
}

func defineLegacyRestApi(router *mux.Router, conf *ServerConfig, db *bolt.DB) error {
	getCollection := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		coll, err := stodb.Read(tx).Collection(mux.Vars(r)["collectionId"])
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

		req := &stotypes.CreateCollectionRequest{}
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

		blobRef, err := stotypes.BlobRefFromHex(mux.Vars(r)["blobRef"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var volumeId int
		if err := db.View(func(tx *bolt.Tx) error {
			coll, err := stodb.Read(tx).Collection(collectionId)
			if err != nil {
				return err
			}

			volumeId = coll.DesiredVolumes[0]

			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := conf.DiskAccess.WriteBlob(volumeId, collectionId, *blobRef, r.Body); err != nil {
			// FIXME: some could be StatusBadRequest
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// shared by getBlob(), getBlobHead()
	getBlobCommon := func(blobRefSerialized string, w http.ResponseWriter) (*stotypes.BlobRef, *stotypes.Blob) {
		blobRef, err := stotypes.BlobRefFromHex(blobRefSerialized)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, nil
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		blobMetadata, err := stodb.Read(tx).Blob(*blobRef)
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

		bestVolumeId, err := conf.DiskAccess.BestVolumeId(blobMetadata.Volumes)
		if err != nil {
			http.Error(w, stotypes.ErrBlobNotAccessibleOnThisNode.Error(), http.StatusInternalServerError)
			return
		}

		file, err := conf.DiskAccess.Fetch(*blobRef, bestVolumeId)
		if err != nil {
			if os.IsNotExist(err) {
				// should not happen, because metadata said that we should have this blob
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if _, err := io.Copy(w, file); err != nil {
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

	return nil
}

func saveNewCollection(parentDirectoryId string, name string, tx *bolt.Tx) (*stotypes.Collection, error) {
	if _, err := stodb.Read(tx).Directory(parentDirectoryId); err != nil {
		if err == blorm.ErrNotFound {
			return nil, errors.New("parent directory not found")
		} else {
			return nil, err
		}
	}

	// TODO: resolve this from closest parent that has policy defined?
	replicationPolicy, err := stodb.Read(tx).ReplicationPolicy("default")
	if err != nil {
		return nil, err
	}

	encryptionKey := [32]byte{}
	if _, err := rand.Read(encryptionKey[:]); err != nil {
		return nil, err
	}

	collection := &stotypes.Collection{
		ID:             stoutils.NewCollectionId(),
		Created:        time.Now(),
		Directory:      parentDirectoryId,
		Name:           name,
		DesiredVolumes: replicationPolicy.DesiredVolumes,
		Head:           stotypes.NoParentId,
		EncryptionKey:  encryptionKey,
		Changesets:     []stotypes.CollectionChangeset{},
		Metadata:       map[string]string{},
	}

	// highly unlikely
	if _, err := stodb.Read(tx).Collection(collection.ID); err != blorm.ErrNotFound {
		return nil, errors.New("accidentally generated duplicate collection ID")
	}

	return collection, stodb.CollectionRepository.Update(collection, tx)
}
