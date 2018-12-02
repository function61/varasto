package bupserver

import (
	"encoding/json"
	"fmt"
	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/function61/bup/pkg/blobdriver"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/gokit/stopper"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
)

type VolumeDriverMap map[string]blobdriver.Driver

func runServer(stop *stopper.Stopper) error {
	defer stop.Done()

	db, err := storm.Open("/tmp/bup.db", storm.Codec(msgpack.Codec))
	if err != nil {
		return err
	}
	defer db.Close()

	serverConfig, err := readConfigFromDatabaseOrBootstrapIfNeeded(db)
	if err != nil {
		return err
	}

	log.Printf(
		"client's auth token %s, server URL http://%s",
		serverConfig.ClientsAuthToken,
		serverConfig.SelfNode.Addr)

	volumeDrivers := VolumeDriverMap{}

	for _, volumeId := range serverConfig.SelfNode.AccessToVolumes {
		// FIXME: use tx here
		var volume buptypes.Volume
		panicIfError(db.One("ID", volumeId, &volume))

		switch volume.Driver {
		case buptypes.VolumeDriverKindLocalFs:
			volumeDrivers[volume.ID] = blobdriver.NewLocalFs(volume.DriverOpts)
		default:
			panic(fmt.Errorf("unsupported volume driver: %s", volume.Driver))
		}
	}

	router := mux.NewRouter()

	if err := defineApi(router, *serverConfig, volumeDrivers, db); err != nil {
		return err
	}

	srv := http.Server{
		Addr:    ":8066",
		Handler: router,
	}

	workers := stopper.NewManager()

	go StartReplicationController(db, volumeDrivers, workers.Stopper())

	go func(stop *stopper.Stopper) {
		defer stop.Done()

		srv.ListenAndServe()
	}(workers.Stopper())

	<-stop.Signal

	if err := srv.Shutdown(nil); err != nil {
		log.Fatalf("Shutdown: %v", err)
	}

	workers.StopAllWorkersAndWait()

	return nil
}

func defineApi(router *mux.Router, conf ServerConfig, volumeDrivers VolumeDriverMap, db *storm.DB) error {
	getCollections := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		var collections []buptypes.Collection
		panicIfError(db.All(&collections))

		outJson(w, collections)
	}

	getCollection := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		collection := &buptypes.Collection{}
		if err := db.One("ID", mux.Vars(r)["collectionId"], collection); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		outJson(w, collection)
	}

	newCollection := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		req := &buptypes.CreateCollectionRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		collection := buptypes.Collection{
			ID:                buputils.NewCollectionId(),
			Name:              req.Name,
			ReplicationPolicy: "default",
			Head:              buptypes.NoParentId,
			Changesets:        []buptypes.CollectionChangeset{},
		}

		panicIfError(db.Save(&collection))

		outJson(w, collection)
	}

	uploadBlob := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		blobRef, err := buptypes.BlobRefFromHex(mux.Vars(r)["blobRef"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		volumeId := conf.SelfNode.AccessToVolumes[0]
		volumeDriver := volumeDrivers[volumeId]

		blobSizeBytes, err := volumeDriver.Store(*blobRef, buputils.BlobHashVerifier(r.Body, *blobRef))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		panicIfError(err)

		// log.Printf("Wrote blob %s for upload %s", blobRef.AsHex(), uploadId)
		log.Printf("Wrote blob %s", blobRef.AsHex())

		fc := buptypes.Blob{
			Ref:        *blobRef,
			Volumes:    []string{volumeId},
			Referenced: false,
		}

		tx, errTxBegin := db.Begin(true)
		panicIfError(errTxBegin)
		defer tx.Rollback()

		var volume buptypes.Volume
		panicIfError(tx.One("ID", volumeId, &volume))

		volume.BlobCount++
		volume.BlobSizeTotal += blobSizeBytes

		panicIfError(tx.Save(&volume))
		panicIfError(tx.Save(&fc))
		panicIfError(tx.Commit())
	}

	commitChangeset := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		collectionId := mux.Vars(r)["collectionId"]

		var changeset buptypes.CollectionChangeset
		panicIfError(json.NewDecoder(r.Body).Decode(&changeset))

		tx, errTxBegin := db.Begin(true)
		panicIfError(errTxBegin)
		defer tx.Rollback()

		var coll buptypes.Collection
		panicIfError(tx.One("ID", collectionId, &coll))

		var replPolicy buptypes.ReplicationPolicy
		panicIfError(tx.One("ID", coll.ReplicationPolicy, &replPolicy))

		if collectionHasChangesetId(changeset.ID, &coll) {
			http.Error(w, "changeset ID already in collection", http.StatusBadRequest)
			return
		}

		if changeset.Parent != buptypes.NoParentId && !collectionHasChangesetId(changeset.Parent, &coll) {
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
				ref, err := buptypes.BlobRefFromHex(refHex)
				if err != nil {
					panic(err)
				}

				var blob buptypes.Blob
				if err := tx.One("Ref", ref, &blob); err != nil {
					http.Error(w, fmt.Sprintf("blob %s not found", ref.AsHex()), http.StatusBadRequest)
					return
				}

				// FIXME: if same changeset mentions same blob many times, we update the old blob
				// metadata many times due to the transaction reads not seeing uncommitted writes
				blob.Referenced = true
				blob.VolumesPendingReplication = missingFromLeftHandSide(
					blob.Volumes,
					replPolicy.DesiredVolumes)
				blob.IsPendingReplication = len(blob.VolumesPendingReplication) > 0

				panicIfError(tx.Save(&blob))
			}
		}

		// update head pointer
		coll.Head = changeset.ID
		coll.Changesets = append(coll.Changesets, changeset)

		panicIfError(tx.Save(&coll))
		panicIfError(tx.Commit())

		log.Printf("Collection %s changeset %s committed", coll.ID, changeset.ID)

		outJson(w, coll)
	}

	getNodes := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		var nodes []buptypes.Node
		panicIfError(db.All(&nodes))
		outJson(w, nodes)
	}

	getVolumes := func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		var volumes []buptypes.Volume
		panicIfError(db.All(&volumes))
		outJson(w, volumes)
	}

	// shared by getBlob(), getBlobHead()
	getBlobCommon := func(blobRefSerialized string, w http.ResponseWriter) (*buptypes.BlobRef, *buptypes.Blob) {
		blobRef, err := buptypes.BlobRefFromHex(blobRefSerialized)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, nil
		}

		blobMetadata := &buptypes.Blob{}
		if err := db.One("Ref", blobRef, blobMetadata); err != nil {
			if err == storm.ErrNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
				return nil, nil
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil, nil
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
			if driver, found := volumeDrivers[volumeId]; found {
				foundDriver = driver
				break
			}
		}

		// TODO: issue HTTP redirect to correct node?
		if foundDriver == nil {
			http.Error(w, buptypes.ErrBlobNotAccessibleOnThisNode.Error(), http.StatusInternalServerError)
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

		if _, err := io.Copy(w, buputils.BlobHashVerifier(file, *blobRef)); err != nil {
			// FIXME: shouldn't try to write headers if even one write went to ResponseWriter
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	router.HandleFunc("/blobs/{blobRef}", getBlob).Methods(http.MethodGet)
	router.HandleFunc("/blobs/{blobRef}", getBlobHead).Methods(http.MethodHead)
	router.HandleFunc("/blobs/{blobRef}", uploadBlob).Methods(http.MethodPost)

	router.HandleFunc("/collections", getCollections).Methods(http.MethodGet)
	router.HandleFunc("/collections", newCollection).Methods(http.MethodPost)
	router.HandleFunc("/collections/{collectionId}", getCollection).Methods(http.MethodGet)
	router.HandleFunc("/collections/{collectionId}/changesets", commitChangeset).Methods(http.MethodPost)

	router.HandleFunc("/nodes", getNodes).Methods(http.MethodGet)

	router.HandleFunc("/volumes", getVolumes).Methods(http.MethodGet)

	router.HandleFunc("/db/export", func(w http.ResponseWriter, r *http.Request) {
		if !authenticate(conf, w, r) {
			return
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		panicIfError(exportDb(tx, w))
	}).Methods(http.MethodGet)

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	// myRouter.Handle('/ui/{rest}', http.StripPrefix("/ui/", http.FileServer(http.Dir("ui/"))))

	return nil
}
