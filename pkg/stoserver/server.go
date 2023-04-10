package stoserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"

	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/pi-security-module/pkg/f61ui"
	"github.com/function61/varasto/pkg/blobstore"
	"github.com/function61/varasto/pkg/blobstore/googledriveblobstore"
	"github.com/function61/varasto/pkg/blobstore/localfsblobstore"
	"github.com/function61/varasto/pkg/blobstore/s3blobstore"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/childprocesscontroller"
	"github.com/function61/varasto/pkg/frontend"
	"github.com/function61/varasto/pkg/gokitbp"
	"github.com/function61/varasto/pkg/logtee"
	"github.com/function61/varasto/pkg/restartcontroller"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stomediascanner"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stokeystore"
	"github.com/function61/varasto/pkg/stoserver/storeplication"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/public"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
)

var (
	defaultDialer = net.Dialer{}
)

type ServerConfigFile struct {
	DbLocation                   string `json:"db_location"`
	DisableReplicationController bool   `json:"disable_replication_controller"`
	DisableMediaScanner          bool   `json:"disable_media_scanner"`
}

func runServer(
	ctx context.Context,
	logger *log.Logger,
	logTail *logtee.StringTail,
	restarter *restartcontroller.Controller,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	confReloader := &configReloader{restarter: restarter}

	logl := logex.Levels(logger)

	scf, err := readServerConfigFile()
	if err != nil {
		return err // has enough context
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

	tasks := taskrunner.New(ctx, logger)

	withErrButWaitTasks := func(err error) error {
		// no harm in double-canceling if context was already canceled
		cancel()

		if errTaskWait := tasks.Wait(); errTaskWait != nil {
			if err != nil {
				return fmt.Errorf("%v; also tasks.Wait(): %v", err, errTaskWait)
			} else {
				return errTaskWait
			}
		} else {
			return err
		}
	}

	// upcoming:
	// - transcoding server
	// - (possible microservice) generic user/auth microservice (pluggable for Lambda-hosted function61 one)
	// - blobstore drivers, so integrity verification jobs can be ionice'd?
	mediascannerSockAddr := "mediascanner.sock"

	serverConfig.MediaScanner = &subsystem{
		id:        stoservertypes.SubsystemIdMediascanner,
		httpMount: "/api/mediascanner",
		enabled:   !scf.DisableMediaScanner,
		controller: childprocesscontroller.New(
			[]string{os.Args[0], "server", stomediascanner.Verb, "--addr", "domainsocket://" + mediascannerSockAddr},
			"Media scanner",
			logex.Prefix("manager(mediascanner)", logger),
			logex.Prefix("mediascanner", logger),
			func(task func(context.Context) error) { tasks.Start("manager(mediascanner)", task) }),
		sockPath: mediascannerSockAddr,
	}

	fuseProjectorSockAddr := "fuseprojector.sock"

	serverConfig.FuseProjector = &subsystem{
		id:        stoservertypes.SubsystemIdFuseProjector,
		httpMount: "/api/fuse",
		enabled:   false,
		controller: childprocesscontroller.New(
			[]string{os.Args[0], "fuse", "serve", "--stop-if-stdin-closes", "--addr=domainsocket://" + fuseProjectorSockAddr},
			"FUSE projector",
			logex.Prefix("manager(fuse)", logger),
			logex.Prefix("fuse", logger),
			func(task func(context.Context) error) { tasks.Start("manager(fuse)", task) }),
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
		func(run func(context.Context) error) { tasks.Start("integrityctrl", run) })

	defineRestApi(
		router,
		serverConfig,
		db,
		ivController,
		mwares,
		logex.Prefix("restapi", logger),
	)

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

	mountSubsystem(serverConfig.MediaScanner)
	mountSubsystem(serverConfig.FuseProjector)

	eventLog, err := createNonPersistingEventLog()
	if err != nil {
		return withErrButWaitTasks(err)
	}
	chandlers := &cHandlers{db, serverConfig, ivController, logger, confReloader} // Bing
	cHandlersInvoker := stoservertypes.CommandInvoker(chandlers)

	registerCommandEndpoints(
		router,
		eventLog,
		cHandlersInvoker,
		mwares)

	schedulerController, err := setupScheduledJobs(
		cHandlersInvoker,
		eventLog,
		db,
		logger,
		func(fn func(context.Context) error) { tasks.Start("scheduler", fn) },
		func(fn func(context.Context) error) { tasks.Start("scheduler/snapshots", fn) })
	if err != nil {
		return withErrButWaitTasks(err)
	}
	serverConfig.Scheduler = schedulerController

	router.Handle(
		"/metrics",
		serverConfig.Metrics.MetricsHttpHandler())

	defineUI(router)

	srv := &http.Server{
		Addr:              "0.0.0.0:443", // 0.0.0.0 = listen on all interfaces
		Handler:           serverConfig.Metrics.WrapHttpServer(router),
		ReadHeaderTimeout: gokitbp.DefaultReadHeaderTimeout,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12, // require TLS 1.2 minimum
			Certificates: []tls.Certificate{serverConfig.TlsCertificate.keypair},
		},
	}

	for _, mount := range serverConfig.ClusterWideMounts {
		// one might disable this during times of massive data ingestion to lessen the read
		// pressure from the initial disk the blobs land on
		if scf.DisableReplicationController {
			continue
		}

		logPrefix := fmt.Sprintf("replctrl/%d", mount.Volume)

		serverConfig.ReplicationControllers[mount.Volume] = storeplication.New(
			mount.Volume,
			db,
			serverConfig.DiskAccess,
			logex.Prefix(logPrefix, logger),
			func(fn func(context.Context) error) { tasks.Start(logPrefix, fn) })
	}

	tasks.Start("listener "+srv.Addr, func(_ context.Context) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServeTLS("", ""))
	})

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	tasks.Start("metricscollector", serverConfig.Metrics.Task(serverConfig, db))

	logl.Info.Printf(
		"node %s (ver. %s) started",
		serverConfig.SelfNodeId,
		dynversion.Version)

	// got cleanly until here. any of the tasks can error however, and that error
	// will be returned here. on graceful shutdown this'll return nil
	return tasks.Wait()
}

func defineUI(router *mux.Router) {
	assetsPath := "/assets"

	publicFiles := http.FileServer(http.FS(public.Content))

	//    "/assets/style.css"
	// => "/style.css"
	router.PathPrefix(assetsPath + "/").Handler(http.StripPrefix(assetsPath+"/", publicFiles))
	router.Handle("/favicon.ico", publicFiles)
	router.Handle("/robots.txt", publicFiles)

	uiHandler := f61ui.IndexHtmlHandler(assetsPath)

	frontend.RegisterUiRoutes(router, uiHandler)
}

type discardEventLog struct{}

func (d *discardEventLog) Append(_ []ehevent.Event) error {
	return nil // ok nom nom
}

func createNonPersistingEventLog() (eventlog.Log, error) {
	return &discardEventLog{}, nil
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
	MediaScanner           *subsystem
	FuseProjector          *subsystem
	TlsCertificate         wrappedKeypair
	KeyStore               *stokeystore.Store
	FailedMountNames       []string
	Metrics                *metricsController
}

// returns blorm.ErrBucketNotFound if bootstrap needed
func readConfigFromDatabase(
	db *bbolt.DB,
	scf *ServerConfigFile,
	logger *log.Logger,
	logTail *logtee.StringTail,
) (*ServerConfig, error) {
	if err := validateSchemaVersionAndMigrateIfNeeded(db, logger); err != nil {
		return nil, err
	}

	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

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

	keyStore, err := loadAndFillKeyStore(tx)
	if err != nil {
		return nil, err
	}

	dam := stodiskaccess.New(&dbbma{db, keyStore})

	failedMountNames := []string{}

	metrics := newMetricsController()

	for _, mount := range clusterWideMounts {
		if mount.Node != selfNode.ID { // only mount vols for our node
			continue
		}

		volume, err := stodb.Read(tx).Volume(mount.Volume)
		if err != nil {
			return nil, err
		}

		originalDriver, err := getDriver(*volume, mount, logger)
		if err != nil {
			return nil, err
		}

		// wrap original driver with metrics-collecting proxy
		driver := metrics.WrapDriver(originalDriver, volume.ID, volume.UUID, volume.Label)

		// for safety. if on Windows we're using external USB disks, their drive letters
		// could get mixed up and we could mount the wrong volume and that would not be great.
		if err := dam.Mount(context.TODO(), volume.ID, volume.UUID, driver); err != nil {
			logex.Levels(logger).Error.Printf("volume %s mount: %v", volume.UUID, err)

			failedMountNames = append(failedMountNames, volume.Label)
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
		KeyStore:               keyStore,
		ReplicationControllers: map[int]*storeplication.Controller{},
		TlsCertificate:         *wrappedKeypair,
		FailedMountNames:       failedMountNames,
		Metrics:                metrics,
	}, nil
}

func getDriver(
	volume stotypes.Volume,
	mount stotypes.VolumeMount,
	logger *log.Logger,
) (blobstore.Driver, error) {
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
		if os.IsNotExist(err) {
			return nil, errors.New("'config.json' not found; did you read installation instructions:\n  https://function61.com/varasto/docs/install/")
		} else { // some other error
			return nil, err
		}
	}

	// TODO: for each "STO_" prefixed, make sure we processed them all to prevent typos
	if os.Getenv("STO_DISABLE_REPLICATION_CONTROLLER") != "" {
		scf.DisableReplicationController = true
	}

	if os.Getenv("STO_DISABLE_MEDIASCANNER") != "" {
		scf.DisableMediaScanner = true
	}

	return scf, nil
}

type dbbma struct {
	db       *bbolt.DB
	keyStore *stokeystore.Store
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

	if err := d.db.View(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(coll)
		if err != nil {
			return fmt.Errorf("collection not found: %v", err)
		}

		// first is special and should always exist. the following are for deduplicated content
		kenv = &coll.EncryptionKeys[0]
		return nil
	}); err != nil {
		return "", nil, err
	}

	dek, err := d.keyStore.DecryptDek(*kenv)
	if err != nil {
		return "", nil, err
	}

	return kenv.KeyId, dek, nil
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
	kenv := stotypes.FindDekEnvelope(blob.EncryptionKeyId, encryptionKeys)
	if kenv == nil {
		return nil, fmt.Errorf("(should not happen) encryption key envelope not found for: %s", ref.AsHex())
	}

	dek, err := d.keyStore.DecryptDek(*kenv)
	if err != nil {
		return nil, err
	}

	return &stodiskaccess.BlobMeta{
		Ref:           ref,
		RealSize:      blob.Size,
		SizeOnDisk:    blob.SizeOnDisk,
		IsCompressed:  blob.IsCompressed,
		EncryptionKey: dek,
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

	return tx.Commit()
}

func (d *dbbma) writeBlobReplicatedInternal(blob *stotypes.Blob, volumeId int, size int64, tx *bbolt.Tx) error {
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

	cert, err := cryptoutil.ParsePemX509Certificate(certPem)
	if err != nil {
		return nil, err
	}

	return &wrappedKeypair{keypair, *cert}, nil
}

// wrapping restarter to implement config reloading by sending (delayed) restart signal
// to the server
type configReloader struct {
	restarter *restartcontroller.Controller
	timerOnce sync.Once
}

func (r *configReloader) ReloadConfig() {
	// protect against multiple timed restarts per single run instance
	go r.timerOnce.Do(func() {
		time.Sleep(3 * time.Second)

		if err := r.restarter.Restart(); err != nil {
			panic(err)
		}
	})
}
