package stoserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/ubackup/pkg/ubbackup"
	"github.com/function61/ubackup/pkg/ubconfig"
	"github.com/function61/ubackup/pkg/ubstorage"
	"github.com/function61/ubackup/pkg/ubtypes"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"go.etcd.io/bbolt"
)

const (
	varastoUbackupServiceID = "varasto"
)

var (
	backupInProgress nonBlockingLock
)

func (c *cHandlers) DatabaseBackupConfigure(cmd *stoservertypes.DatabaseBackupConfigure, ctx *command.Ctx) error {
	serializedUbConfig, err := json.Marshal(&[]string{
		cmd.Bucket,
		cmd.BucketRegion,
		cmd.AccessKeyId,
		cmd.AccessKeySecret,
		cmd.EncryptionPublicKey,
		cmd.AlertmanagerBaseUrl,
		cmd.EncryptionPrivateKeyStorageLocationDescription,
	})
	if err != nil {
		return err
	}

	conf, err := parseSerializedUbConfig(serializedUbConfig)
	if err != nil {
		return err
	}

	// validates bucket, region, access key {id,secret}
	if cmd.ConnectivityCheck {
		if _, err := listUbackupStoredBackups(conf.Storage, c.logger); err != nil {
			return err
		}
	}

	return c.setConfigValue(stodb.CfgUbackupConfig, string(serializedUbConfig))
}

func (c *cHandlers) DatabaseBackup(cmd *stoservertypes.DatabaseBackup, ctx *command.Ctx) error {
	conf, err := ubConfigFromDB(c.db)
	if err != nil {
		return err
	}

	ok, unlock := backupInProgress.TryLock()
	if !ok {
		return errors.New("another backup already in progress")
	}
	defer unlock()

	backup := ubtypes.BackupForTarget(ubtypes.BackupTarget{
		ServiceName: varastoUbackupServiceID,
		TaskId:      fmt.Sprintf("%d", os.Getpid()),
		Snapshotter: ubtypes.CustomStream(func(snapshotSink io.Writer) error {
			tx, err := c.db.Begin(false)
			if err != nil {
				return err
			}
			defer func() { ignoreError(tx.Rollback()) }()

			return stodbimportexport.Export(tx, snapshotSink)
		}),
	})

	return ubbackup.BackupAndStore(ctx.Ctx, backup, *conf, logex.Prefix("µbackup", c.logger))
}

func ubConfigFromDB(db *bbolt.DB) (*ubconfig.Config, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	serializedUbConfig, err := stodb.CfgUbackupConfig.GetRequired(tx)
	if err != nil {
		return nil, err
	}

	return parseSerializedUbConfig([]byte(serializedUbConfig))
}

func parseSerializedUbConfig(serializedUbConfig []byte) (*ubconfig.Config, error) {
	parts := []string{}

	if err := json.Unmarshal(serializedUbConfig, &parts); err != nil {
		return nil, err
	}

	// encryptionPrivateKeyStorageLocationDescription was added later (though not used in backend so it's not read here)
	if partCount := len(parts); partCount != 6 && partCount != 7 {
		return nil, fmt.Errorf("unexpected number of parts: %d", partCount)
	}

	return &ubconfig.Config{
		EncryptionPublicKey: parts[4],
		Storage: ubconfig.StorageConfig{
			S3: &ubconfig.StorageS3Config{
				Bucket:          parts[0],
				BucketRegion:    parts[1],
				AccessKeyId:     parts[2],
				AccessKeySecret: parts[3],
			},
		},
		AlertManager: &ubconfig.AlertManagerConfig{
			BaseUrl: parts[5],
		},
	}, nil
}

func listUbackupStoredBackups(
	storageConf ubconfig.StorageConfig,
	logger *log.Logger,
) ([]stoservertypes.UbackupStoredBackup, error) {
	storage, err := ubstorage.StorageFromConfig(storageConf, logger)
	if err != nil {
		return nil, err
	}

	backups, err := storage.List(varastoUbackupServiceID)
	if err != nil {
		return nil, err
	}

	ret := []stoservertypes.UbackupStoredBackup{}

	for _, backup := range backups {
		ret = append(ret, stoservertypes.UbackupStoredBackup{
			ID:          backup.ID,
			Size:        int(backup.Size),
			Timestamp:   backup.Timestamp,
			Description: backup.Description,
		})
	}

	return ret, nil
}

func downloadBackup(
	backupID string,
	output io.Writer,
	conf ubconfig.Config,
	logger *log.Logger,
) error {
	storage, err := ubstorage.StorageFromConfig(conf.Storage, logger)
	if err != nil {
		return err
	}

	backupReader, err := storage.Get(backupID)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, backupReader); err != nil {
		return err
	}

	return backupReader.Close()
}
