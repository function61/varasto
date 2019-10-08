package stoserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/ubackup/pkg/ubbackup"
	"github.com/function61/ubackup/pkg/ubconfig"
	"github.com/function61/ubackup/pkg/ubstorage"
	"github.com/function61/ubackup/pkg/ubtypes"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"go.etcd.io/bbolt"
	"io"
	"log"
	"os"
	"time"
)

const (
	varastoUbackupServiceId = "varasto"
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

	target := ubtypes.BackupTarget{
		ServiceName: varastoUbackupServiceId,
		TaskId:      fmt.Sprintf("%d", os.Getpid()),
	}

	// do this in another thread, because this is going to take a while
	go func() {
		log := logex.Prefix("µbackup", c.logger)
		logl := logex.Levels(log)

		logl.Info.Println("starting")

		backup := ubtypes.BackupForTarget(target)

		if err := ubbackup.BackupAndStore(context.TODO(), backup, *conf, func(sink io.Writer) error {
			tx, err := c.db.Begin(false)
			if err != nil {
				return err
			}
			defer tx.Rollback()

			return stodbimportexport.Export(tx, sink)
		}, log); err != nil {
			logl.Error.Printf("failed: %v", err)
		} else {
			logl.Info.Println("success")

			if err := markBackupComplete(backup.Started, c.db); err != nil {
				logl.Error.Printf("markBackupComplete: %v", err)
			}
		}
	}()

	return nil
}

func markBackupComplete(timestamp time.Time, db *bolt.DB) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := stohealth.UpdateMetadatabackupLastSuccess(timestamp, tx); err != nil {
		return err
	}

	return tx.Commit()
}

func ubConfigFromDb(db *bolt.DB) (*ubconfig.Config, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
