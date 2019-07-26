package stoserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/logex"
	"github.com/function61/ubackup/pkg/ubbackup"
	"github.com/function61/ubackup/pkg/ubtypes"
	"io"
	"os"
)

func (c *cHandlers) DatabaseBackup(cmd *DatabaseBackup, ctx *command.Ctx) error {
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

		if err := ubbackup.BackupAndStore(context.TODO(), target, *c.conf.File.BackupConfig, func(sink io.Writer) error {
			tx, err := c.db.Begin(false)
			if err != nil {
				return err
			}
			defer tx.Rollback()

			return exportDb(tx, sink)
		}, log); err != nil {
			logl.Error.Printf("failed: %v", err)
		} else {
			logl.Info.Println("success")
		}
	}()

	return nil
}
