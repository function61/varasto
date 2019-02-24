package varastoserver

import (
	"fmt"
	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type ServerConfigFile struct {
	DbLocation     string `json:"db_location"`
	AllowBootstrap bool   `json:"allow_bootstrap"`
}

func runServer(logger *log.Logger, stop *stopper.Stopper) error {
	defer stop.Done()

	logl := logex.Levels(logger)

	scf := &ServerConfigFile{}
	if err := jsonfile.Read("config.json", &scf, true); err != nil {
		return err
	}

	db, err := storm.Open(scf.DbLocation, storm.Codec(msgpack.Codec))
	if err != nil {
		return err
	}
	defer db.Close()

	serverConfig, err := readConfigFromDatabase(db, logger)
	if err != nil { // maybe need bootstrap?
		// totally unexpected error?
		if err != storm.ErrNotFound {
			return err
		}

		if !scf.AllowBootstrap {
			logl.Error.Fatalln("bootstrap needed but AllowBootstrap false")
		}

		// was not found error => run bootstrap
		if err := bootstrap(db, logex.Prefix("bootstrap", logger)); err != nil {
			return err
		}

		serverConfig, err = readConfigFromDatabase(db, logger)
		if err != nil {
			return err
		}
	} else {
		if scf.AllowBootstrap {
			logl.Error.Fatalln("AllowBootstrap true after bootstrap already done => dangerous")
		}
	}

	mwares := createDummyMiddlewares(serverConfig)

	router := mux.NewRouter()

	if err := defineRestApi(router, serverConfig, db, mwares, logex.Prefix("restapi", logger)); err != nil {
		return err
	}

	eventLog, err := createNonPersistingEventLog()
	if err != nil {
		return err
	}

	registerCommandEndpoints(
		router,
		eventLog,
		&cHandlers{db, serverConfig},
		mwares)

	if err := defineUi(router); err != nil {
		return err
	}

	srv := http.Server{
		Addr:    ":8066",
		Handler: router,
	}

	workers := stopper.NewManager()

	go StartReplicationController(
		db,
		serverConfig,
		logex.Prefix("replicationcontroller", logger),
		workers.Stopper())

	go func(stop *stopper.Stopper) {
		defer stop.Done()

		srv.ListenAndServe()
	}(workers.Stopper())

	logl.Info.Printf(
		"node %s (ver. %s) started",
		serverConfig.SelfNode.ID,
		dynversion.Version)

	<-stop.Signal

	if err := srv.Shutdown(nil); err != nil {
		logl.Error.Fatalf("Shutdown: %v", err)
	}

	workers.StopAllWorkersAndWait()

	return nil
}

type VolumeDriverByVolumeId map[int]blobdriver.Driver

type ServerConfig struct {
	SelfNode          varastotypes.Node
	ClusterWideMounts map[int]varastotypes.VolumeMount
	VolumeDrivers     VolumeDriverByVolumeId // only for mounts on self node
	ClientsAuthTokens map[string]bool
}

func readConfigFromDatabase(db *storm.DB, logger *log.Logger) (*ServerConfig, error) {
	var nodeId string
	if err := db.Get("settings", "nodeId", &nodeId); err != nil {
		return nil, err
	}

	selfNode, err := QueryWithTx(db).Node(nodeId)
	if err != nil {
		return nil, err
	}

	myMounts := []varastotypes.VolumeMount{}
	if err := db.Find("Node", selfNode.ID, &myMounts); err != nil && err != storm.ErrNotFound {
		return nil, err
	}

	clusterWideMounts := []varastotypes.VolumeMount{}
	if err := db.All(&clusterWideMounts); err != nil {
		return nil, err
	}

	clusterWideMountsMapped := map[int]varastotypes.VolumeMount{}
	for _, mv := range clusterWideMounts {
		clusterWideMountsMapped[mv.Volume] = mv
	}

	clients := []varastotypes.Client{}
	if err := db.All(&clients); err != nil {
		return nil, err
	}

	authTokens := map[string]bool{}
	for _, client := range clients {
		authTokens[client.AuthToken] = true
	}

	volumeDrivers := VolumeDriverByVolumeId{}

	for _, mountedVolume := range myMounts {
		volume, err := QueryWithTx(db).Volume(mountedVolume.Volume)
		if err != nil {
			return nil, err
		}

		volumeDrivers[mountedVolume.Volume] = getDriver(*volume, mountedVolume, logger)
	}

	return &ServerConfig{
		SelfNode:          *selfNode,
		ClusterWideMounts: clusterWideMountsMapped,
		VolumeDrivers:     volumeDrivers,
		ClientsAuthTokens: authTokens,
	}, nil
}

func getDriver(volume varastotypes.Volume, mount varastotypes.VolumeMount, logger *log.Logger) blobdriver.Driver {
	switch mount.Driver {
	case varastotypes.VolumeDriverKindLocalFs:
		return blobdriver.NewLocalFs(
			volume.UUID,
			mount.DriverOpts,
			logex.Prefix("blobdriver/localfs", logger))
	default:
		panic(fmt.Errorf("unsupported volume driver: %s", mount.Driver))
	}
}
