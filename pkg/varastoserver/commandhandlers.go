package varastoserver

import (
	"fmt"
	"github.com/asdine/storm"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"github.com/gorilla/mux"
	"net/http"
)

// we are currently using the command pattern very wrong!
type cHandlers struct {
	db   *storm.DB
	conf *ServerConfig
}

func (c *cHandlers) VolumeCreate(cmd *VolumeCreate, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	allVolumes := []varastotypes.Volume{}
	panicIfError(c.db.All(&allVolumes))

	if err := tx.Save(&varastotypes.Volume{
		ID:    len(allVolumes) + 1,
		UUID:  varastoutils.NewVolumeUuid(),
		Label: cmd.Name,
		Quota: int64(1024 * 1024 * cmd.Quota),
	}); err != nil {
		return err
	}

	return tx.Commit()
}

// FIXME: name ends in 2 because conflicts with types.VolumeMount
func (c *cHandlers) VolumeMount2(cmd *VolumeMount2, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	vol, err := QueryWithTx(tx).Volume(cmd.Id)
	if err != nil {
		return err
	}

	// TODO: grab driver instance by this spec?
	mountSpec := &varastotypes.VolumeMount{
		ID:         varastoutils.NewVolumeMountId(),
		Volume:     vol.ID,
		Node:       c.conf.SelfNode.ID,
		Driver:     varastotypes.VolumeDriverKindLocalFs,
		DriverOpts: cmd.DriverOpts,
	}

	// try mounting the volume
	mount := blobdriver.NewLocalFs(vol.UUID, mountSpec.DriverOpts, nil)
	if err := mount.Mountable(); err != nil {
		return err
	}

	if err := tx.Save(mountSpec); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *cHandlers) VolumeUnmount(cmd *VolumeUnmount, ctx *command.Ctx) error {
	tx, err := c.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	mount, err := QueryWithTx(tx).VolumeMount(cmd.Id)
	if err != nil {
		return err
	}

	if err := tx.DeleteStruct(mount); err != nil {
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

	if err := tx.Save(&varastotypes.Directory{
		ID:     varastoutils.NewDirectoryId(),
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

	collections := []varastotypes.Collection{}
	if err := tx.Find("Directory", dir.ID, &collections); err != nil && err != storm.ErrNotFound {
		return err
	}

	subDirs := []varastotypes.Directory{}
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

	if err := tx.Save(&varastotypes.Client{
		ID:        varastoutils.NewClientId(),
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

	if err := tx.DeleteStruct(&varastotypes.Client{
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
