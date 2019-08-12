package stoserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/ubackup/pkg/ubbackup"
	"github.com/function61/ubackup/pkg/ubtypes"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"go.etcd.io/bbolt"
	"io"
	"os"
	"time"
)

func (c *cHandlers) DatabaseBackup(cmd *stoservertypes.DatabaseBackup, ctx *command.Ctx) error {
	if c.conf.File.BackupConfig == nil {
		return errors.New("backups not configured")
	}

	target := ubtypes.BackupTarget{
		ServiceName: "varasto",
		TaskId:      fmt.Sprintf("%d", os.Getpid()),
	}

	// do this in another thread, because this is going to take a while
	go func() {
		log := logex.Prefix("µbackup", c.logger)
		logl := logex.Levels(log)

		logl.Info.Println("starting")

		backup := ubtypes.BackupForTarget(target)

		if err := ubbackup.BackupAndStore(context.TODO(), backup, *c.conf.File.BackupConfig, func(sink io.Writer) error {
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
