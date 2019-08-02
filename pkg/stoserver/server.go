package stoserver

import (
	"fmt"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/stopper"
	"github.com/function61/ubackup/pkg/ubconfig"
	"github.com/function61/varasto/pkg/blobstore"
	"github.com/function61/varasto/pkg/blobstore/googledriveblobstore"
	"github.com/function61/varasto/pkg/blobstore/localfsblobstore"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/storeplication"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"log"
	"net/http"
	"os"
)

type ServerConfigFile struct {
	DbLocation                   string           `json:"db_location"`
	AllowBootstrap               bool             `json:"allow_bootstrap"`
	DisableReplicationController bool             `json:"disable_replication_controller"`
	BackupConfig                 *ubconfig.Config `json:"backup_config"`
}

func runServer(logger *log.Logger, stop *stopper.Stopper) error {
	defer stop.Done()

	logl := logex.Levels(logger)

	scf, err := readServerConfigFile()
	if err != nil {
		return err
	}

	db, err := stodb.Open(scf.DbLocation)
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
		if err := stodb.Bootstrap(db, logex.Prefix("bootstrap", logger)); err != nil {
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

	ivController := stointegrityverifier.NewController(
		db,
		stodb.IntegrityVerificationJobRepository,
		stodb.BlobRepository,
		serverConfig.DiskAccess,
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
		&cHandlers{db, serverConfig, ivController, logger},
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
		go storeplication.StartReplicationController(
			db,
			serverConfig.DiskAccess,
			logex.Prefix("replctrl", logger),
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

type ServerConfig struct {
	File              ServerConfigFile
	SelfNode          stotypes.Node
	ClusterWideMounts map[int]stotypes.VolumeMount
	DiskAccess        *stodiskaccess.Controller // only for mounts on self node
	ClientsAuthTokens map[string]bool
}

// returns blorm.ErrNotFound if bootstrap needed
func readConfigFromDatabase(db *bolt.DB, scf *ServerConfigFile, logger *log.Logger) (*ServerConfig, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	nodeId, err := stodb.CfgNodeId.GetRequired(tx)
	if err != nil {
		return nil, err
	}

	selfNode, err := stodb.Read(tx).Node(nodeId)
	if err != nil {
		return nil, err
	}

	clusterWideMounts := []stotypes.VolumeMount{}
	if err := stodb.VolumeMountRepository.Each(stodb.VolumeMountAppender(&clusterWideMounts), tx); err != nil {
		return nil, err
	}

	myMounts := []stotypes.VolumeMount{}
	for _, mount := range clusterWideMounts {
		if mount.Node == selfNode.ID {
			myMounts = append(myMounts, mount)
		}
	}

	clusterWideMountsMapped := map[int]stotypes.VolumeMount{}
	for _, mv := range clusterWideMounts {
		clusterWideMountsMapped[mv.Volume] = mv
	}

	clients := []stotypes.Client{}
	if err := stodb.ClientRepository.Each(stodb.ClientAppender(&clients), tx); err != nil {
		return nil, err
	}

	authTokens := map[string]bool{}
	for _, client := range clients {
		authTokens[client.AuthToken] = true
	}

	dam := stodiskaccess.New(&dbbma{db})

	for _, mountedVolume := range myMounts {
		volume, err := stodb.Read(tx).Volume(mountedVolume.Volume)
		if err != nil {
			return nil, err
		}

		driver, err := getDriver(*volume, mountedVolume, logger)
		if err != nil {
			return nil, err
		}

		// for safety. if on Windows we're using external USB disks, their drive letters
		// could get mixed up and we could mount the wrong volume and that would not be great.
		if err := driver.Mountable(); err != nil {
			logex.Levels(logger).Error.Printf("Volume %s not mountable: %v", volume.UUID, err)
		} else {
			if mountedVolume.ID == "xQbd" { // FIXME
				dam.Define(volume.ID, driver, false)
			} else {
				dam.Define(volume.ID, driver, true)
			}
		}
	}

	return &ServerConfig{
		File:              *scf,
		SelfNode:          *selfNode,
		ClusterWideMounts: clusterWideMountsMapped,
		DiskAccess:        dam,
		ClientsAuthTokens: authTokens,
	}, nil
}

func getDriver(volume stotypes.Volume, mount stotypes.VolumeMount, logger *log.Logger) (blobstore.Driver, error) {
	switch mount.Driver {
	case stotypes.VolumeDriverKindLocalFs:
		return localfsblobstore.New(
			volume.UUID,
			mount.DriverOpts,
			logex.Prefix("blobdriver/localfs", logger)), nil
	case stotypes.VolumeDriverKindGoogleDrive:
		return googledriveblobstore.New(
			mount.DriverOpts,
			logex.Prefix("blobdriver/googledrive", logger))
	default:
		return nil, fmt.Errorf("unsupported volume driver: %s", mount.Driver)
	}
}

func readServerConfigFile() (*ServerConfigFile, error) {
	scf := &ServerConfigFile{}
	if err := jsonfile.Read("config.json", &scf, true); err != nil {
		return nil, err
	}

	return scf, nil
}

type dbbma struct {
	db *bolt.DB
}

func (d *dbbma) QueryCollectionEncryptionKey(coll string) ([]byte, error) {
	var result []byte

	if err := d.db.View(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(coll)
		if err != nil {
			return fmt.Errorf("collection not found: %v", err)
		}

		result = coll.EncryptionKey[:]

		return nil
	}); err != nil {
		if err == blorm.ErrNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	return result, nil
}

func (d *dbbma) QueryBlobMetadata(ref stotypes.BlobRef) (*stodiskaccess.BlobMeta, error) {
	tx, err := d.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blob, err := stodb.Read(tx).Blob(ref)
	if err != nil {
		if err == blorm.ErrNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	coll, err := stodb.Read(tx).Collection(blob.Coll)
	if err != nil {
		return nil, fmt.Errorf("collection %s not found: %v", blob.Coll, err)
	}

	return &stodiskaccess.BlobMeta{
		Ref:                 ref,
		RealSize:            blob.Size,
		SizeOnDisk:          blob.SizeOnDisk,
		IsCompressed:        blob.IsCompressed,
		EncryptionKey:       coll.EncryptionKey[:],
		EncryptionKeyOfColl: blob.Coll,
		ExpectedCrc32:       blob.Crc32,
	}, nil
}

func (d *dbbma) WriteBlobReplicated(meta *stodiskaccess.BlobMeta, volumeId int) error {
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	blobToUpdate, err := stodb.Read(tx).Blob(meta.Ref)
	if err != nil {
		return err
	}

	// FIXME: this is temporary to fix my own data
	if len(blobToUpdate.Crc32) == 0 {
		updateBlobFromMeta(meta, blobToUpdate)
	}

	// saves Blob and Volume
	if err := d.writeBlobReplicatedInternal(blobToUpdate, volumeId, int64(meta.SizeOnDisk), tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *dbbma) WriteBlobCreated(meta *stodiskaccess.BlobMeta, volumeId int) error {
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newBlob := &stotypes.Blob{
		Ref:        meta.Ref,
		Volumes:    []int{},
		Referenced: false, // this will be set to true on commit
	}

	updateBlobFromMeta(meta, newBlob)

	// writes Volumes & VolumesPendingReplication
	if err := d.writeBlobReplicatedInternal(newBlob, volumeId, int64(meta.SizeOnDisk), tx); err != nil {
		return err
	}

	log.Printf("wrote blob %s", meta.Ref.AsHex())

	return tx.Commit()
}

func (d *dbbma) writeBlobReplicatedInternal(blob *stotypes.Blob, volumeId int, size int64, tx *bolt.Tx) error {
	if sliceutil.ContainsInt(blob.Volumes, volumeId) {
		return fmt.Errorf(
			"race condition: someone already replicated %s to %d",
			blob.Ref.AsHex(),
			volumeId)
	}

	blob.Volumes = append(blob.Volumes, volumeId)

	// remove succesfully replicated volume from pending list
	blob.VolumesPendingReplication = sliceutil.FilterInt(blob.VolumesPendingReplication, func(volId int) bool {
		return volId != volumeId
	})

	if err := stodb.BlobRepository.Update(blob, tx); err != nil {
		return err
	}

	volume, err := stodb.Read(tx).Volume(volumeId)
	if err != nil {
		return err
	}

	volume.BlobCount++
	volume.BlobSizeTotal += size

	if err := stodb.VolumeRepository.Update(volume, tx); err != nil {
		return err
	}

	return nil
}

func updateBlobFromMeta(meta *stodiskaccess.BlobMeta, blob *stotypes.Blob) {
	blob.Coll = meta.EncryptionKeyOfColl
	blob.IsCompressed = meta.IsCompressed
	blob.Size = meta.RealSize
	blob.SizeOnDisk = meta.SizeOnDisk
	blob.Crc32 = meta.ExpectedCrc32
}