package bupserver

import (
	"fmt"
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/buputils"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/httpauth"
	"github.com/gorilla/mux"
	"net/http"
)

// we are currently using the command pattern very wrong!
type cHandlers struct {
	db *storm.DB
}

func (c *cHandlers) VolumeCreate(cmd *VolumeCreate, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	allVolumes := []buptypes.Volume{}
	panicIfError(c.db.All(&allVolumes))

	if err := tx.Save(&buptypes.Volume{
		ID:    len(allVolumes) + 1,
		UUID:  buputils.NewVolumeUuid(),
		Label: cmd.Name,
		Quota: int64(1024 * 1024 * cmd.Quota),
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) DirectoryCreate(cmd *DirectoryCreate, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Save(&buptypes.Directory{
		ID:     buputils.NewDirectoryId(),
		Parent: cmd.Parent,
		Name:   cmd.Name,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) DirectoryDelete(cmd *DirectoryDelete, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dir, err := QueryWithTx(tx).Directory(cmd.Id)
	if err != nil {
		return err
	}

	collections := []buptypes.Collection{}
	if err := tx.Find("Directory", dir.ID, &collections); err != nil && err != storm.ErrNotFound {
		return err
	}

	subDirs := []buptypes.Directory{}
	if err := tx.Find("Parent", dir.ID, &subDirs); err != nil && err != storm.ErrNotFound {
		return err
	}

	if len(collections) > 0 {
		return fmt.Errorf("Cannot delete directory because it has %d collection(s)", len(collections))
	}

	if len(subDirs) > 0 {
		return fmt.Errorf("Cannot delete directory because it has %d directory(s)", len(subDirs))
	}

	if err := tx.DeleteStruct(dir); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) DirectoryRename(cmd *DirectoryRename, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dir, err := QueryWithTx(tx).Directory(cmd.Id)
	if err != nil {
		return err
	}

	dir.Name = cmd.Name

	if err := tx.Save(dir); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) CollectionMove(cmd *CollectionMove, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// check for existence
	_, err = QueryWithTx(tx).Directory(cmd.Directory)
	if err != nil {
		return err
	}

	coll, err := QueryWithTx(tx).Collection(cmd.Collection)
	if err != nil {
		return err
	}

	coll.Directory = cmd.Directory

	if err := tx.Save(coll); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) CollectionRename(cmd *CollectionRename, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	coll, err := QueryWithTx(tx).Collection(cmd.Collection)
	if err != nil {
		return err
	}

	coll.Name = cmd.Name

	if err := tx.Save(coll); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) ClientCreate(cmd *ClientCreate, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Save(&buptypes.Client{
		ID:        buputils.NewClientId(),
		Name:      cmd.Name,
		AuthToken: cryptorandombytes.Base64Url(32),
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) ClientRemove(cmd *ClientRemove, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.DeleteStruct(&buptypes.Client{
		ID: cmd.Id,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func registerCommandEndpoints(
	router *mux.Router,
	eventLog eventlog.Log,
	cmdHandlers CommandHandlers,
	mwares httpauth.MiddlewareChainMap,
) {
	router.HandleFunc("/command/{commandName}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commandName := mux.Vars(r)["commandName"]

		httpErr := httpcommand.Serve(
			w,
			r,
			mwares,
			commandName,
			Allocators,
			cmdHandlers,
			eventLog)
		if httpErr != nil {
			if !httpErr.ErrorResponseAlreadySentByMiddleware() {
				http.Error(
					w,
					httpErr.ErrorCode+": "+httpErr.Description,
					httpErr.StatusCode) // making many assumptions here
			}
		} else {
			// no-op => ok
			w.Write([]byte(`{}`))
		}
	})).Methods(http.MethodPost)
}
