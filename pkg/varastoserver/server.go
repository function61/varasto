package varastoserver

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/varastoserver/varastointegrityverifier"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"log"
	"net/http"
)

type ServerConfigFile struct {
	DbLocation                   string `json:"db_location"`
	BackupPath                   string `json:"backup_path"`
	AllowBootstrap               bool   `json:"allow_bootstrap"`
	DisableReplicationController bool   `json:"disable_replication_controller"`
	TheMovieDbApiKey             string `json:"themoviedb_apikey"`
}

func runServer(logger *log.Logger, stop *stopper.Stopper) error {
	defer stop.Done()

	logl := logex.Levels(logger)

	scf, err := readServerConfigFile()
	if err != nil {
		return err
	}

	db, err := boltOpen(scf)
	if err != nil {
		return err
	}
	defer db.Close()

	serverConfig, err := readConfigFromDatabase(db, scf, logger)
	if err != nil { // maybe need bootstrap?
		// totally unexpected error?
		if err != blorm.ErrNotFound {
			return err
		}

		if !scf.AllowBootstrap {
			logl.Error.Fatalln("bootstrap needed but AllowBootstrap false")
		}

		// was not found error => run bootstrap
		if err := bootstrap(db, logex.Prefix("bootstrap", logger)); err != nil {
			return err
		}

		serverConfig, err = readConfigFromDatabase(db, scf, logger)
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

	workers := stopper.NewManager()

	ivController := varastointegrityverifier.NewController(
		db,
		IntegrityVerificationJobRepository,
		BlobRepository,
		serverConfig.VolumeDrivers,
		logex.Prefix("integrityctrl", logger),
		workers.Stopper())

	if err := defineRestApi(
		router,
		serverConfig,
		db,
		ivController,
		mwares,
		logex.Prefix("restapi", logger),
	); err != nil {
		return err
	}

	eventLog, err := createNonPersistingEventLog()
	if err != nil {
		return err
	}

	registerCommandEndpoints(
		router,
		eventLog,
		&cHandlers{db, serverConfig, ivController},
		mwares)

	if err := defineUi(router); err != nil {
		return err
	}

	srv := http.Server{
		Addr:    "0.0.0.0:8066", // 0.0.0.0 = listen on all interfaces
		Handler: router,
	}

	// one might disable this during times of massive data ingestion to lessen the read
	// pressure from the initial disk the blobs land on
	if !scf.DisableReplicationController {
		go StartReplicationController(
			db,
			serverConfig,
			logex.Prefix("replicationcontroller", logger),
			workers.Stopper())
	}

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
	File              ServerConfigFile
	SelfNode          varastotypes.Node
	ClusterWideMounts map[int]varastotypes.VolumeMount
	VolumeDrivers     VolumeDriverByVolumeId // only for mounts on self node
	ClientsAuthTokens map[string]bool
}

// returns ErrNotFound if bootstrap needed
func readConfigFromDatabase(db *bolt.DB, scf *ServerConfigFile, logger *log.Logger) (*ServerConfig, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	configBucket := tx.Bucket(configBucketKey)
	if configBucket == nil {
		return nil, blorm.ErrNotFound
	}

	nodeId := string(configBucket.Get(configBucketNodeKey))
	if nodeId == "" {
		return nil, errors.New("config bucket node ID not found")
	}

	selfNode, err := QueryWithTx(tx).Node(nodeId)
	if err != nil {
		return nil, err
	}

	clusterWideMounts := []varastotypes.VolumeMount{}
	if err := VolumeMountRepository.Each(volumeMountAppender(&clusterWideMounts), tx); err != nil {
		return nil, err
	}

	myMounts := []varastotypes.VolumeMount{}
	for _, mount := range clusterWideMounts {
		if mount.Node == selfNode.ID {
			myMounts = append(myMounts, mount)
		}
	}

	clusterWideMountsMapped := map[int]varastotypes.VolumeMount{}
	for _, mv := range clusterWideMounts {
		clusterWideMountsMapped[mv.Volume] = mv
	}

	clients := []varastotypes.Client{}
	if err := ClientRepository.Each(clientAppender(&clients), tx); err != nil {
		return nil, err
	}

	authTokens := map[string]bool{}
	for _, client := range clients {
		authTokens[client.AuthToken] = true
	}

	volumeDrivers := VolumeDriverByVolumeId{}

	for _, mountedVolume := range myMounts {
		volume, err := QueryWithTx(tx).Volume(mountedVolume.Volume)
		if err != nil {
			return nil, err
		}

		driver := getDriver(*volume, mountedVolume, logger)

		// for safety. if on Windows we're using external USB disks, their drive letters
		// could get mixed up and we could mount the wrong volume and that would not be great.
		if err := driver.Mountable(); err != nil {
			logex.Levels(logger).Error.Printf("Volume %s not mountable: %v", volume.UUID, err)
		} else {
			volumeDrivers[mountedVolume.Volume] = driver
		}
	}

	return &ServerConfig{
		File:              *scf,
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

func readServerConfigFile() (*ServerConfigFile, error) {
	scf := &ServerConfigFile{}
	if err := jsonfile.Read("config.json", &scf, true); err != nil {
		return nil, err
	}

	return scf, nil
}

func boltOpen(scf *ServerConfigFile) (*bolt.DB, error) {
	return bolt.Open(scf.DbLocation, 0700, nil)
}
