package stoserver

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/stopper"
	"github.com/function61/pi-security-module/pkg/extractpublicfiles"
	"github.com/function61/varasto/pkg/blobstore"
	"github.com/function61/varasto/pkg/blobstore/googledriveblobstore"
	"github.com/function61/varasto/pkg/blobstore/localfsblobstore"
	"github.com/function61/varasto/pkg/blobstore/s3blobstore"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/childprocesscontroller"
	"github.com/function61/varasto/pkg/logtee"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/storeplication"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

var (
	defaultDialer = net.Dialer{}
)

type ServerConfigFile struct {
	DbLocation                   string `json:"db_location"`
	DisableReplicationController bool   `json:"disable_replication_controller"`
}

func runServer(logger *log.Logger, logTail *logtee.StringTail, stop *stopper.Stopper) error {
	defer stop.Done()

	logl := logex.Levels(logger)

	// if public.tar.gz is not present in our working directory, try to download & extract it automatically
	if err := extractpublicfiles.Run(extractpublicfiles.BintrayDownloadUrl(
		"function61",
		"dl",
		"varasto/"+dynversion.Version+"/"+extractpublicfiles.PublicFilesArchiveFilename), extractpublicfiles.PublicFilesArchiveFilename, logger); err != nil {
		return err
	}

	scf, err := readServerConfigFile()
	if err != nil {
		return err
	}

	db, err := stodb.Open(scf.DbLocation)
	if err != nil {
		return err
	}
	defer db.Close()

	serverConfig, err := readConfigFromDatabase(db, scf, logger, logTail)
	if err != nil { // maybe need bootstrap?
		// totally unexpected error?
		if err != blorm.ErrBucketNotFound {
			return err
		}

		// was not found error => run bootstrap
		if err := stodb.Bootstrap(db, logex.Prefix("bootstrap", logger)); err != nil {
			return err
		}

		serverConfig, err = readConfigFromDatabase(db, scf, logger, logTail)
		if err != nil {
			return err
		}
	}

	workers := stopper.NewManager()

	// upcoming:
	// - transcoding server
	// - (possible microservice) generic user/auth microservice (pluggable for Lambda-hosted function61 one)
	// - blobstore drivers, so integrity verification jobs can be ionice'd?
	thumbnailerSockAddr := "/tmp/sto-thumbnailer.sock"

	serverConfig.ThumbServer = &subsystem{
		id:        stoservertypes.SubsystemIdThumbnailGenerator,
		httpMount: "/api/thumbnails",
		enabled:   true,
		controller: childprocesscontroller.New(
			[]string{os.Args[0], "server", "thumbserver", "--addr", "domainsocket://" + thumbnailerSockAddr},
			"Thumbnail generator",
			logex.Prefix("manager(thumbserver)", logger),
			logex.Prefix("thumbserver", logger),
			workers.Stopper()),
		sockPath: thumbnailerSockAddr,
	}

	fuseProjectorSockAddr := "/tmp/sto-fuseprojector.sock"

	serverConfig.FuseProjector = &subsystem{
		id:        stoservertypes.SubsystemIdFuseProjector,
		httpMount: "/api/fuse",
		enabled:   false,
		controller: childprocesscontroller.New(
			[]string{os.Args[0], "fuse", "serve", "--addr", "domainsocket://" + fuseProjectorSockAddr},
			"FUSE projector",
			logex.Prefix("manager(fuse)", logger),
			logex.Prefix("fuse", logger),
			workers.Stopper()),
		sockPath: fuseProjectorSockAddr,
	}

	mwares := createDummyMiddlewares(serverConfig)

	router := mux.NewRouter()

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

	mountSubsystem := func(subsys *subsystem) {
		router.PathPrefix(subsys.httpMount).Handler(&httputil.ReverseProxy{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return defaultDialer.DialContext(ctx, "unix", subsys.sockPath)
				},
			},
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				// does not matter with domain sockets unless the server uses
				// name-based hosting (does not in this case)
				req.URL.Host = "server"
			},
		})

		if subsys.enabled {
			subsys.controller.Start()
		}
	}

	mountSubsystem(serverConfig.ThumbServer)
	mountSubsystem(serverConfig.FuseProjector)

	eventLog, err := createNonPersistingEventLog()
	if err != nil {
		return err
	}
	chandlers := &cHandlers{db, serverConfig, ivController, logger} // Bing

	registerCommandEndpoints(
		router,
		eventLog,
		chandlers,
		mwares)

	schedulerController, err := setupScheduledJobs(
		chandlers,
		eventLog,
		db,
		logger,
		workers.Stopper(),
		workers.Stopper())
	if err != nil {
		return err
	}
	serverConfig.Scheduler = schedulerController

	if err := defineUi(router); err != nil {
		return err
	}

	srv := http.Server{
		Addr:    "0.0.0.0:4486", // 0.0.0.0 = listen on all interfaces
		Handler: router,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{serverConfig.TlsCertificate.keypair},
		},
	}

	for _, mount := range serverConfig.ClusterWideMounts {
		// one might disable this during times of massive data ingestion to lessen the read
		// pressure from the initial disk the blobs land on
		if scf.DisableReplicationController {
			continue
		}

		serverConfig.ReplicationControllers[mount.Volume] = storeplication.Start(
			mount.Volume,
			db,
			serverConfig.DiskAccess,
			logex.Prefix(fmt.Sprintf("replctrl/%d", mount.Volume), logger),
			workers.Stopper())
	}

	go func(stop *stopper.Stopper) {
		defer stop.Done()

		if err := srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			logl.Error.Fatalf("ListenAndServe: %v", err)
		}
	}(workers.Stopper())

	logl.Info.Printf(
		"node %s (ver. %s) started",
		serverConfig.SelfNodeId,
		dynversion.Version)

	<-stop.Signal

	if err := srv.Shutdown(context.TODO()); err != nil {
		logl.Error.Fatalf("Shutdown: %v", err)
	}

	workers.StopAllWorkersAndWait()

	return nil
}

// pairs together a subprocess controller and its socket path
type subsystem struct {
	id         stoservertypes.SubsystemId
	enabled    bool
	httpMount  string
	controller *childprocesscontroller.Controller
	sockPath   string
}

type ServerConfig struct {
	File                   ServerConfigFile
	SelfNodeId             string
	SelfNodeSmartBackend   stoservertypes.SmartBackend
	ClusterWideMounts      map[int]stotypes.VolumeMount
	DiskAccess             *stodiskaccess.Controller // only for mounts on self node
	ClientsAuthTokens      map[string]bool
	LogTail                *logtee.StringTail
	ReplicationControllers map[int]*storeplication.Controller
	Scheduler              *scheduler.Controller
	ThumbServer            *subsystem
	FuseProjector          *subsystem
	TlsCertificate         wrappedKeypair
}

// returns blorm.ErrBucketNotFound if bootstrap needed
func readConfigFromDatabase(db *bolt.DB, scf *ServerConfigFile, logger *log.Logger, logTail *logtee.StringTail) (*ServerConfig, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	if err := stodb.ValidateSchemaVersion(tx); err != nil {
		return nil, err
	}

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

	keksParsed := map[string]*rsa.PrivateKey{}

	keks := []stotypes.KeyEncryptionKey{}
	if err := stodb.KeyEncryptionKeyRepository.Each(stodb.KeyEncryptionKeyAppender(&keks), tx); err != nil {
		return nil, err
	}

	for _, kek := range keks {
		privateKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey(strings.NewReader(kek.PrivateKey))
		if err != nil {
			return nil, err
		}

		keksParsed[kek.Fingerprint] = privateKey
	}

	dam := stodiskaccess.New(&dbbma{db, keksParsed})

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
		if err := dam.Mount(context.TODO(), volume.ID, volume.UUID, driver); err != nil {
			logex.Levels(logger).Error.Printf("volume %s mount: %v", volume.UUID, err)
		}
	}

	tlsCertKey, err := stodb.CfgNodeTlsCertKey.GetRequired(tx)
	if err != nil {
		return nil, err
	}

	wrappedKeypair, err := mkWrappedKeypair([]byte(selfNode.TlsCert), []byte(tlsCertKey))
	if err != nil {
		return nil, err
	}

	return &ServerConfig{
		File:                   *scf,
		SelfNodeId:             selfNode.ID,
		SelfNodeSmartBackend:   selfNode.SmartBackend,
		ClusterWideMounts:      clusterWideMountsMapped,
		DiskAccess:             dam,
		ClientsAuthTokens:      authTokens,
		LogTail:                logTail,
		ReplicationControllers: map[int]*storeplication.Controller{},
		TlsCertificate:         *wrappedKeypair,
	}, nil
}

func getDriver(volume stotypes.Volume, mount stotypes.VolumeMount, logger *log.Logger) (blobstore.Driver, error) {
	switch stoservertypes.VolumeDriverKindExhaustive42cc85(mount.Driver) {
	case stoservertypes.VolumeDriverKindLocalFs:
		return localfsblobstore.New(
			volume.UUID,
			mount.DriverOpts,
			logex.Prefix("blobdriver/localfs", logger)), nil
	case stoservertypes.VolumeDriverKindAwsS3:
		return s3blobstore.New(
			mount.DriverOpts,
			logex.Prefix("blobdriver/s3", logger))
	case stoservertypes.VolumeDriverKindGoogledrive:
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
	db   *bolt.DB
	keks map[string]*rsa.PrivateKey
}

func (d *dbbma) QueryBlobExists(ref stotypes.BlobRef) (bool, error) {
	tx, err := d.db.Begin(false)
	if err != nil {
		return false, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	if _, err := stodb.Read(tx).Blob(ref); err != nil {
		if err == blorm.ErrNotFound {
			return false, nil
		}

		return false, err // some other error
	}

	return true, nil
}

func (d *dbbma) QueryCollectionEncryptionKeyForNewBlobs(coll string) (string, []byte, error) {
	var kenv *stotypes.KeyEnvelope

	if err := d.db.View(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(coll)
		if err != nil {
			return fmt.Errorf("collection not found: %v", err)
		}

		// first should always exist
		kenv = &coll.EncryptionKeys[0]
		return nil
	}); err != nil {
		return "", nil, err
	}

	// search for the first key slot that we have a decryption key for
	for _, slot := range kenv.Slots {
		decrypionKey, found := d.keks[slot.KekFingerprint]
		if !found {
			continue
		}

		encryptionKey, err := stotypes.DecryptKek(*kenv, decrypionKey)
		if err != nil {
			return "", nil, err
		}

		return kenv.KeyId, encryptionKey, nil
	}

	return "", nil, fmt.Errorf("no key decryption key found for key id %s", kenv.KeyId)
}

func (d *dbbma) QueryBlobCrc32(ref stotypes.BlobRef) ([]byte, error) {
	tx, err := d.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	blob, err := stodb.Read(tx).Blob(ref)
	if err != nil {
		if err == blorm.ErrNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	return blob.Crc32, nil
}

func (d *dbbma) QueryBlobMetadata(ref stotypes.BlobRef, encryptionKeys []stotypes.KeyEnvelope) (*stodiskaccess.BlobMeta, error) {
	tx, err := d.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	blob, err := stodb.Read(tx).Blob(ref)
	if err != nil {
		if err == blorm.ErrNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	// we are given a list of key envelopes. the first one is for new blobs written to this
	// collection, and the following are for when blobs get deduplicated into this collection -
	// source collection's key envelopes are copied into target collection's key envelopes
	kenv := stotypes.FindKeyById(blob.EncryptionKeyId, encryptionKeys)
	if kenv == nil {
		return nil, fmt.Errorf("(should not happen) encryption key envelope not found for: %s", ref.AsHex())
	}

	var kdc *rsa.PrivateKey
	for _, slot := range kenv.Slots {
		var found bool
		kdc, found = d.keks[slot.KekFingerprint]
		if found {
			break
		}
	}

	if kdc == nil {
		return nil, fmt.Errorf("no decryption key found for key %s", kenv.KeyId)
	}

	encryptionKey, err := stotypes.DecryptKek(*kenv, kdc)
	if err != nil {
		return nil, err
	}

	return &stodiskaccess.BlobMeta{
		Ref:           ref,
		RealSize:      blob.Size,
		SizeOnDisk:    blob.SizeOnDisk,
		IsCompressed:  blob.IsCompressed,
		EncryptionKey: encryptionKey,
		ExpectedCrc32: blob.Crc32,
	}, nil
}

func (d *dbbma) WriteBlobReplicated(ref stotypes.BlobRef, volumeId int) error {
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	blobToUpdate, err := stodb.Read(tx).Blob(ref)
	if err != nil {
		return err
	}

	// saves Blob and Volume
	if err := d.writeBlobReplicatedInternal(blobToUpdate, volumeId, int64(blobToUpdate.SizeOnDisk), tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *dbbma) WriteBlobCreated(meta *stodiskaccess.BlobMeta, volumeId int) error {
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	newBlob := &stotypes.Blob{
		Ref:             meta.Ref,
		Volumes:         []int{}, // writeBlobReplicatedInternal() adds this
		Referenced:      false,   // this will be set to true on commit
		EncryptionKeyId: meta.EncryptionKeyId,
		IsCompressed:    meta.IsCompressed,
		Size:            meta.RealSize,
		SizeOnDisk:      meta.SizeOnDisk,
		Crc32:           meta.ExpectedCrc32,
	}

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

// for some reason tls.Certificate doesn't have cert in parsed form. ".Leaf" would be it,
// but it's documented as nil with successful X509KeyPair()
type wrappedKeypair struct {
	keypair tls.Certificate
	cert    x509.Certificate
}

func mkWrappedKeypair(certPem, keyPem []byte) (*wrappedKeypair, error) {
	keypair, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, err
	}

	cert, err := cryptoutil.ParsePemX509Certificate(bytes.NewBuffer(certPem))
	if err != nil {
		return nil, err
	}

	return &wrappedKeypair{keypair, *cert}, nil
}
