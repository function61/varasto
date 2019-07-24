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
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/varastofuse/varastofuseclient"
	"github.com/function61/varasto/pkg/varastoserver/stodb"
	"github.com/function61/varasto/pkg/varastoserver/varastointegrityverifier"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"net/http"
	"strings"
)

// we are currently using the command pattern very wrong!
type cHandlers struct {
	db           *bolt.DB
	conf         *ServerConfig
	ivController *varastointegrityverifier.Controller
}

func (c *cHandlers) VolumeCreate(cmd *VolumeCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		max := 0

		allVolumes := []varastotypes.Volume{}
		if err := stodb.VolumeRepository.Each(stodb.VolumeAppender(&allVolumes), tx); err != nil {
			return err
		}

		for _, vol := range allVolumes {
			if vol.ID > max {
				max = vol.ID
			}
		}

		return stodb.VolumeRepository.Update(&varastotypes.Volume{
			ID:    max + 1,
			UUID:  varastoutils.NewVolumeUuid(),
			Label: cmd.Name,
			Quota: mebibytesToBytes(cmd.Quota),
		}, tx)
	})
}

func (c *cHandlers) VolumeChangeQuota(cmd *VolumeChangeQuota, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Quota = mebibytesToBytes(cmd.Quota)

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeDescription(cmd *VolumeChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Description = cmd.Description

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

// FIXME: name ends in 2 because conflicts with types.VolumeMount
func (c *cHandlers) VolumeMount2(cmd *VolumeMount2, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		mountSpec := &varastotypes.VolumeMount{
			ID:         varastoutils.NewVolumeMountId(),
			Volume:     vol.ID,
			Node:       c.conf.SelfNode.ID,
			Driver:     varastotypes.VolumeDriverKind(cmd.Kind),
			DriverOpts: cmd.DriverOpts,
		}

		// try mounting the volume
		driver, err := getDriver(*vol, *mountSpec, logex.Discard)
		if err != nil {
			return err
		}

		if err := driver.Mountable(); err != nil {
			return err
		}

		return stodb.VolumeMountRepository.Update(mountSpec, tx)
	})
}

func (c *cHandlers) VolumeUnmount(cmd *VolumeUnmount, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		mount, err := stodb.Read(tx).VolumeMount(cmd.Id)
		if err != nil {
			return err
		}

		return stodb.VolumeMountRepository.Delete(mount, tx)
	})
}

func (c *cHandlers) VolumeVerifyIntegrity(cmd *VolumeVerifyIntegrity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		job := &varastotypes.IntegrityVerificationJob{
			ID:       varastoutils.NewIntegrityVerificationJobId(),
			Started:  ctx.Meta.Timestamp,
			VolumeId: cmd.Id,
		}

		return stodb.IntegrityVerificationJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) DirectoryCreate(cmd *DirectoryCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.DirectoryRepository.Update(
			varastotypes.NewDirectory(
				varastoutils.NewDirectoryId(),
				cmd.Parent,
				cmd.Name),
			tx)
	})
}

func (c *cHandlers) DirectoryDelete(cmd *DirectoryDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		collections, err := stodb.Read(tx).CollectionsByDirectory(dir.ID)
		if err != nil {
			return err
		}

		subDirs, err := stodb.Read(tx).SubDirectories(dir.ID)
		if err != nil {
			return err
		}

		if len(collections) > 0 {
			return fmt.Errorf("Cannot delete directory because it has %d collection(s)", len(collections))
		}

		if len(subDirs) > 0 {
			return fmt.Errorf("Cannot delete directory because it has %d directory(s)", len(subDirs))
		}

		return stodb.DirectoryRepository.Delete(dir, tx)
	})
}

func (c *cHandlers) DirectoryRename(cmd *DirectoryRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Name = cmd.Name

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeDescription(cmd *DirectoryChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Description = cmd.Description

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeSensitivity(cmd *DirectoryChangeSensitivity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		if err := validateSensitivity(cmd.Sensitivity); err != nil {
			return err
		}

		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Sensitivity = cmd.Sensitivity

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryMove(cmd *DirectoryMove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dirToMove, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		// verify that new parent exists
		newParent, err := stodb.Read(tx).Directory(cmd.Directory)
		if err != nil {
			return err
		}

		if dirToMove.ID == newParent.ID {
			return errors.New("dir cannot be its own parent, dawg")
		}

		dirToMove.Parent = newParent.ID

		return stodb.DirectoryRepository.Update(dirToMove, tx)
	})
}

func (c *cHandlers) CollectionCreate(cmd *CollectionCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		_, err := saveNewCollection(cmd.ParentDir, cmd.Name, tx)
		return err
	})
}

func (c *cHandlers) CollectionChangeSensitivity(cmd *CollectionChangeSensitivity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		if err := validateSensitivity(cmd.Sensitivity); err != nil {
			return err
		}

		coll, err := stodb.Read(tx).Collection(cmd.Id)
		if err != nil {
			return err
		}

		coll.Sensitivity = cmd.Sensitivity

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionMove(cmd *CollectionMove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		// check for existence
		if _, err := stodb.Read(tx).Directory(cmd.Directory); err != nil {
			return err
		}

		// Collection is validated as non-empty
		collIds := strings.Split(cmd.Collection, ",")

		for _, collId := range collIds {
			coll, err := stodb.Read(tx).Collection(collId)
			if err != nil {
				return err
			}

			coll.Directory = cmd.Directory

			if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *cHandlers) CollectionChangeDescription(cmd *CollectionChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Description = cmd.Description

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionRename(cmd *CollectionRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Name = cmd.Name

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionFuseMount(cmd *CollectionFuseMount, ctx *command.Ctx) error {
	vstofuse := varastofuseclient.New()

	if cmd.UnmountOthers {
		if err := vstofuse.UnmountAll(); err != nil {
			return err
		}
	}

	return vstofuse.Mount(cmd.Collection)
}

func (c *cHandlers) CollectionDelete(cmd *CollectionDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		if cmd.Name != coll.Name {
			return fmt.Errorf("repeated name incorrect, expecting %s", coll.Name)
		}

		return stodb.CollectionRepository.Delete(coll, tx)
	})
}

func (c *cHandlers) ClientCreate(cmd *ClientCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.ClientRepository.Update(&varastotypes.Client{
			ID:        varastoutils.NewClientId(),
			Name:      cmd.Name,
			AuthToken: cryptorandombytes.Base64Url(32),
		}, tx)
	})
}

func (c *cHandlers) ClientRemove(cmd *ClientRemove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.ClientRepository.Delete(&varastotypes.Client{
			ID: cmd.Id,
		}, tx)
	})
}

func (c *cHandlers) IntegrityverificationjobResume(cmd *IntegrityverificationjobResume, ctx *command.Ctx) error {
	c.ivController.Resume(cmd.JobId)

	return nil
}

func (c *cHandlers) IntegrityverificationjobStop(cmd *IntegrityverificationjobStop, ctx *command.Ctx) error {
	c.ivController.Stop(cmd.JobId)

	return nil
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
			if _, err := stodb.Read(tx).Volume(desiredVolume); err != nil {
				return fmt.Errorf("desiredVolume %d: %v", desiredVolume, err)
			}
		}

		policy, err := stodb.Read(tx).ReplicationPolicy(cmd.Id)
		if err != nil {
			return err
		}

		policy.DesiredVolumes = desiredVolumes

		return stodb.ReplicationPolicyRepository.Update(policy, tx)
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

func validateSensitivity(in int) error {
	if in < 0 || in > 2 {
		return fmt.Errorf("sensitivity needs to be between 0-2; was %d", in)
	}

	return nil
}
