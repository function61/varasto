package stoserver

import (
	"encoding/json"
	"errors"
	"fmt"
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
	"io"
	"log"
	"os"
)

const (
	varastoUbackupServiceId = "varasto"
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
		if _, err := listUbackupStoredBackups(*conf, c.logger); err != nil {
			return err
		}
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.CfgUbackupConfig.Set(string(serializedUbConfig), tx)
	})
}

func (c *cHandlers) DatabaseBackup(cmd *stoservertypes.DatabaseBackup, ctx *command.Ctx) error {
	conf, err := ubConfigFromDb(c.db)
	if err != nil {
		return err
	}

	ok, unlock := backupInProgress.TryLock()
	if !ok {
		return errors.New("another backup already in progress")
	}
	defer unlock()

	target := ubtypes.BackupTarget{
		ServiceName: varastoUbackupServiceId,
		TaskId:      fmt.Sprintf("%d", os.Getpid()),
	}

	backup := ubtypes.BackupForTarget(target)

	return ubbackup.BackupAndStore(ctx.Ctx, backup, *conf, func(sink io.Writer) error {
		tx, err := c.db.Begin(false)
		if err != nil {
			return err
		}
		defer ignoreError(tx.Rollback())

		return stodbimportexport.Export(tx, sink)
	}, logex.Prefix("µbackup", c.logger))
}

func ubConfigFromDb(db *bolt.DB) (*ubconfig.Config, error) {
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

	if len(parts) != 6 {
		return nil, fmt.Errorf("unexpected number of parts: %d", len(parts))
	}

	return &ubconfig.Config{
		Bucket:              parts[0],
		BucketRegion:        parts[1],
		AccessKeyId:         parts[2],
		AccessKeySecret:     parts[3],
		EncryptionPublicKey: parts[4],
		AlertmanagerBaseUrl: parts[5],
	}, nil
}

func listUbackupStoredBackups(conf ubconfig.Config, logger *log.Logger) ([]stoservertypes.UbackupStoredBackup, error) {
	storage, err := ubstorage.StorageFromConfig(conf, logger)
	if err != nil {
		return nil, err
	}

	backups, err := storage.List(varastoUbackupServiceId)
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
	backupId string,
	output io.Writer,
	conf ubconfig.Config,
	logger *log.Logger,
) error {
	storage, err := ubstorage.StorageFromConfig(conf, logger)
	if err != nil {
		return err
	}

	backupReader, err := storage.Get(backupId)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, backupReader); err != nil {
		return err
	}

	return backupReader.Close()
}
