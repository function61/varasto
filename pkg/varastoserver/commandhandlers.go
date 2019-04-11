package varastoserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/varastofuse/varastofuseclient"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"net/http"
	"strings"
)

// we are currently using the command pattern very wrong!
type cHandlers struct {
	db   *bolt.DB
	conf *ServerConfig
}

func (c *cHandlers) VolumeCreate(cmd *VolumeCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		allVolumes := []varastotypes.Volume{}
		if err := VolumeRepository.Each(volumeAppender(&allVolumes), tx); err != nil {
			return err
		}

		return VolumeRepository.Update(&varastotypes.Volume{
			ID:    len(allVolumes) + 1,
			UUID:  varastoutils.NewVolumeUuid(),
			Label: cmd.Name,
			Quota: mebibytesToBytes(cmd.Quota),
		}, tx)
	})
}

func (c *cHandlers) VolumeChangeQuota(cmd *VolumeChangeQuota, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := QueryWithTx(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Quota = mebibytesToBytes(cmd.Quota)

		return VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeDescription(cmd *VolumeChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := QueryWithTx(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Description = cmd.Description

		return VolumeRepository.Update(vol, tx)
	})
}

// FIXME: name ends in 2 because conflicts with types.VolumeMount
func (c *cHandlers) VolumeMount2(cmd *VolumeMount2, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
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

		return VolumeMountRepository.Update(mountSpec, tx)
	})
}

func (c *cHandlers) VolumeUnmount(cmd *VolumeUnmount, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		mount, err := QueryWithTx(tx).VolumeMount(cmd.Id)
		if err != nil {
			return err
		}

		return VolumeMountRepository.Delete(mount, tx)
	})
}

func (c *cHandlers) DirectoryCreate(cmd *DirectoryCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return DirectoryRepository.Update(&varastotypes.Directory{
			ID:     varastoutils.NewDirectoryId(),
			Parent: cmd.Parent,
			Name:   cmd.Name,
		}, tx)
	})
}

func (c *cHandlers) DirectoryDelete(cmd *DirectoryDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := QueryWithTx(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		collections, err := QueryWithTx(tx).CollectionsByDirectory(dir.ID)
		if err != nil {
			return err
		}

		subDirs, err := QueryWithTx(tx).SubDirectories(dir.ID)
		if err != nil {
			return err
		}

		if len(collections) > 0 {
			return fmt.Errorf("Cannot delete directory because it has %d collection(s)", len(collections))
		}

		if len(subDirs) > 0 {
			return fmt.Errorf("Cannot delete directory because it has %d directory(s)", len(subDirs))
		}

		return DirectoryRepository.Delete(dir, tx)
	})
}

func (c *cHandlers) DirectoryRename(cmd *DirectoryRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := QueryWithTx(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Name = cmd.Name

		return DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeDescription(cmd *DirectoryChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := QueryWithTx(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Description = cmd.Description

		return DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeSensitivity(cmd *DirectoryChangeSensitivity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := QueryWithTx(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Sensitivity = cmd.Sensitivity

		return DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryMove(cmd *DirectoryMove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dirToMove, err := QueryWithTx(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		// verify that new parent exists
		newParent, err := QueryWithTx(tx).Directory(cmd.Directory)
		if err != nil {
			return err
		}

		if dirToMove.ID == newParent.ID {
			return errors.New("dir cannot be its own parent, dawg")
		}

		dirToMove.Parent = newParent.ID

		return DirectoryRepository.Update(dirToMove, tx)
	})
}

func (c *cHandlers) CollectionCreate(cmd *CollectionCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		_, err := saveNewCollection(cmd.ParentDir, cmd.Name, tx)
		return err
	})
}

func (c *cHandlers) CollectionMove(cmd *CollectionMove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		// check for existence
		if _, err := QueryWithTx(tx).Directory(cmd.Directory); err != nil {
			return err
		}

		// Collection is validated as non-empty
		collIds := strings.Split(cmd.Collection, ",")

		for _, collId := range collIds {
			coll, err := QueryWithTx(tx).Collection(collId)
			if err != nil {
				return err
			}

			coll.Directory = cmd.Directory

			if err := CollectionRepository.Update(coll, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *cHandlers) CollectionChangeDescription(cmd *CollectionChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := QueryWithTx(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Description = cmd.Description

		return CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionRename(cmd *CollectionRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := QueryWithTx(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Name = cmd.Name

		return CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionFuseMount(cmd *CollectionFuseMount, ctx *command.Ctx) error {
	return varastofuseclient.New().Mount(cmd.Collection)
}

func (c *cHandlers) CollectionDelete(cmd *CollectionDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := QueryWithTx(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		if cmd.Name != coll.Name {
			return fmt.Errorf("repeated name incorrect, expecting %s", coll.Name)
		}

		return CollectionRepository.Delete(coll, tx)
	})
}

func (c *cHandlers) ClientCreate(cmd *ClientCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return ClientRepository.Update(&varastotypes.Client{
			ID:        varastoutils.NewClientId(),
			Name:      cmd.Name,
			AuthToken: cryptorandombytes.Base64Url(32),
		}, tx)
	})
}

func (c *cHandlers) ClientRemove(cmd *ClientRemove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return ClientRepository.Delete(&varastotypes.Client{
			ID: cmd.Id,
		}, tx)
	})
}

func (c *cHandlers) ReplicationpolicyChangeDesiredVolumes(cmd *ReplicationpolicyChangeDesiredVolumes, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		desiredVolumes := []int{}
		if err := json.Unmarshal([]byte(cmd.DesiredVolumes), &desiredVolumes); err != nil {
			return err
		}

		if len(desiredVolumes) < 1 {
			return errors.New("need at least 1 volume")
		}

		// verify that each volume exists
		for _, desiredVolume := range desiredVolumes {
			if _, err := QueryWithTx(tx).Volume(desiredVolume); err != nil {
				return fmt.Errorf("desiredVolume %d: %v", desiredVolume, err)
			}
		}

		policy, err := QueryWithTx(tx).ReplicationPolicy(cmd.Id)
		if err != nil {
			return err
		}

		policy.DesiredVolumes = desiredVolumes

		return ReplicationPolicyRepository.Update(policy, tx)
	})
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

func mebibytesToBytes(mebibytes int) int64 {
	return int64(mebibytes * 1024 * 1024)
}
