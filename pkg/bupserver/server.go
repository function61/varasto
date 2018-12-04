package bupserver

import (
	"fmt"
	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/function61/bup/pkg/blobdriver"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type VolumeDriverMap map[string]blobdriver.Driver

func runServer(logger *log.Logger, stop *stopper.Stopper) error {
	defer stop.Done()

	logl := logex.Levels(logger)

	db, err := storm.Open("/tmp/bup.db", storm.Codec(msgpack.Codec))
	if err != nil {
		return err
	}
	defer db.Close()

	serverConfig, err := readConfigFromDatabaseOrBootstrapIfNeeded(
		db,
		logex.Prefix("bootstrap", logger))
	if err != nil {
		return err
	}

	logl.Info.Printf(
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
			volumeDrivers[volume.ID] = blobdriver.NewLocalFs(
				volume.DriverOpts,
				logex.Prefix("blobdriver/localfs", logger))
		default:
			panic(fmt.Errorf("unsupported volume driver: %s", volume.Driver))
		}
	}

	router := mux.NewRouter()

	if err := defineRestApi(router, *serverConfig, volumeDrivers, db, logex.Prefix("restapi", logger)); err != nil {
		return err
	}

	if err := defineUi(router, db); err != nil {
		return err
	}

	srv := http.Server{
		Addr:    ":8066",
		Handler: router,
	}

	workers := stopper.NewManager()

	go StartReplicationController(
		db,
		volumeDrivers,
		logex.Prefix("replicationcontroller", logger),
		workers.Stopper())

	go func(stop *stopper.Stopper) {
		defer stop.Done()

		srv.ListenAndServe()
	}(workers.Stopper())

	<-stop.Signal

	if err := srv.Shutdown(nil); err != nil {
		logl.Error.Fatalf("Shutdown: %v", err)
	}

	workers.StopAllWorkersAndWait()

	return nil
}
