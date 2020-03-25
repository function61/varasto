package stodb

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/sslca"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

// opens BoltDB database
func Open(dbLocation string) (*bbolt.DB, error) {
	return bbolt.Open(dbLocation, 0700, nil)
}

func Bootstrap(db *bbolt.DB, logger *log.Logger) error {
	logl := logex.Levels(logger)

	bootstrapTimestamp := time.Now()

	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	// be extra safe and scan the DB to see that it is totally empty
	if err := tx.ForEach(func(name []byte, _ *bbolt.Bucket) error {
		return fmt.Errorf("DB not empty, found bucket: %s", name)
	}); err != nil {
		return err
	}

	if err := BootstrapRepos(tx); err != nil {
		return err
	}

	hostname := "localhost"

	privKeyPem, err := sslca.GenEcP256PrivateKeyPem()
	if err != nil {
		return err
	}

	certPem, err := sslca.SelfSignedServerCert(hostname, "Varasto self-signed", privKeyPem)
	if err != nil {
		return err
	}

	// if we're not inside Docker, we need to use SMART via Docker image because its
	// automation friendly JSON interface is not in mainstream OSes yet
	smartBackend := stoservertypes.SmartBackendSmartCtlViaDocker
	if maybeRunningInsideDocker() {
		// when we're in Docker, we guess we're using the official Varasto image which
		// has the exact correct version of smartctl and so we can invoke it directly
		smartBackend = stoservertypes.SmartBackendSmartCtl
	}

	newNode := &stotypes.Node{
		ID:           stoutils.NewNodeId(),
		Addr:         "https://" + hostname,
		Name:         "dev",
		TlsCert:      string(certPem),
		SmartBackend: smartBackend,
	}

	logl.Info.Printf("generated nodeId: %s", newNode.ID)

	systemAuthToken := stoutils.NewApiKeySecret()

	results := []error{
		NodeRepository.Update(newNode, tx),
		DirectoryRepository.Update(stotypes.NewDirectory(
			"root",
			"",
			"root",
			string(stoservertypes.DirectoryTypeGeneric)), tx),
		VolumeRepository.Update(&stotypes.Volume{
			ID:         1,
			UUID:       stoutils.NewVolumeUuid(),
			Label:      "Default volume",
			Technology: string(stoservertypes.VolumeTechnologyDiskHdd),
			Quota:      1 * 1024 * 1024 * 1024,
		}, tx),
		ReplicationPolicyRepository.Update(&stotypes.ReplicationPolicy{
			ID:             "default",
			Name:           "Default replication policy",
			DesiredVolumes: []int{1},
		}, tx),
		ClientRepository.Update(&stotypes.Client{
			ID:        stoutils.NewClientId(),
			Created:   bootstrapTimestamp,
			Name:      "System",
			AuthToken: systemAuthToken,
		}, tx),
		ScheduledJobRepository.Update(&stotypes.ScheduledJob{
			ID:          "ocKgpTHU3Sk",
			Description: "SMART poller",
			Schedule:    "@every 5m",
			Kind:        stoservertypes.ScheduledJobKindSmartpoll,
			Enabled:     true,
		}, tx),
		ScheduledJobRepository.Update(&stotypes.ScheduledJob{
			ID:          "h-cPYsYtFzM",
			Description: "Metadata backup",
			Schedule:    "@midnight",
			Kind:        stoservertypes.ScheduledJobKindMetadatabackup,
			Enabled:     true,
		}, tx),
		CfgNodeId.Set(newNode.ID, tx),
		CfgNodeTlsCertKey.Set(string(privKeyPem), tx),
	}

	if err := allOk(results); err != nil {
		return err
	}

	if err := configureClientConfig(systemAuthToken); err != nil {
		return err
	}

	return tx.Commit()
}

func BootstrapRepos(tx *bbolt.Tx) error {
	if err := writeSchemaVersion(tx); err != nil {
		return err
	}

	for _, repo := range RepoByRecordType {
		if err := repo.Bootstrap(tx); err != nil {
			return err
		}
	}

	return nil
}

func configureClientConfig(authToken string) error {
	return stoclient.WriteConfig(&stoclient.ClientConfig{
		ServerAddr: "https://localhost",
		AuthToken:  authToken,
		// FuseMountPath: "...",
		TlsInsecureSkipValidation: true, // localhost address, no worries
	})
}

func allOk(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func ignoreError(err error) {
	// no-op
}

// if false, we might not be running in Docker (also any error)
// if true, we are most probably running in Docker
func maybeRunningInsideDocker() bool {
	// https://stackoverflow.com/a/20012536
	initCgroups, err := ioutil.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	return strings.Contains(string(initCgroups), "docker")
}
