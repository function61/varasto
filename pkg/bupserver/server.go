package bupserver

import (
	"fmt"
	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/function61/bup/pkg/blobdriver"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/stopper"
	"github.com/gorilla/mux"
	"log"
	"net/http"
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
