package stoserver

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stofuse/stofuseclient"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"log"
	"net/http"
	"strings"
	"time"
)

// we are currently using the command pattern very wrong!
type cHandlers struct {
	db           *bolt.DB
	conf         *ServerConfig
	ivController *stointegrityverifier.Controller
	logger       *log.Logger
}

func (c *cHandlers) VolumeCreate(cmd *stoservertypes.VolumeCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		max := 0

		allVolumes := []stotypes.Volume{}
		if err := stodb.VolumeRepository.Each(stodb.VolumeAppender(&allVolumes), tx); err != nil {
			return err
		}

		for _, vol := range allVolumes {
			if vol.ID > max {
				max = vol.ID
			}
		}

		return stodb.VolumeRepository.Update(&stotypes.Volume{
			ID:    max + 1,
			UUID:  stoutils.NewVolumeUuid(),
			Label: cmd.Name,
			Quota: mebibytesToBytes(cmd.Quota),
		}, tx)
	})
}

func (c *cHandlers) VolumeChangeQuota(cmd *stoservertypes.VolumeChangeQuota, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Quota = mebibytesToBytes(cmd.Quota)

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeDescription(cmd *stoservertypes.VolumeChangeDescription, ctx *command.Ctx) error {
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
func (c *cHandlers) VolumeMount2(cmd *stoservertypes.VolumeMount2, ctx *command.Ctx) error {
	sameVolumeOnSameNode := func(a, b *stotypes.VolumeMount) bool {
		return a.Volume == b.Volume && a.Node == b.Node
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		mountSpec := &stotypes.VolumeMount{
			ID:         stoutils.NewVolumeMountId(),
			Volume:     vol.ID,
			Node:       c.conf.SelfNode.ID,
			Driver:     stotypes.VolumeDriverKind(cmd.Kind),
			DriverOpts: cmd.DriverOpts,
		}

		allMounts := []stotypes.VolumeMount{}
		if err := stodb.VolumeMountRepository.Each(stodb.VolumeMountAppender(&allMounts), tx); err != nil {
			return err
		}

		for _, otherMount := range allMounts {
			if sameVolumeOnSameNode(mountSpec, &otherMount) {
				return fmt.Errorf("same volume is already mounted at specified node. mount id: %s", otherMount.ID)
			}
		}

		// try mounting the volume
		driver, err := getDriver(*vol, *mountSpec, logex.Discard)
		if err != nil {
			return err
		}

		if err := driver.Mountable(context.TODO()); err != nil {
			return err
		}

		return stodb.VolumeMountRepository.Update(mountSpec, tx)
	})
}

func (c *cHandlers) VolumeUnmount(cmd *stoservertypes.VolumeUnmount, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		mount, err := stodb.Read(tx).VolumeMount(cmd.Id)
		if err != nil {
			return err
		}

		return stodb.VolumeMountRepository.Delete(mount, tx)
	})
}

func (c *cHandlers) VolumeVerifyIntegrity(cmd *stoservertypes.VolumeVerifyIntegrity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		job := &stotypes.IntegrityVerificationJob{
			ID:       stoutils.NewIntegrityVerificationJobId(),
			Started:  ctx.Meta.Timestamp,
			VolumeId: cmd.Id,
		}

		return stodb.IntegrityVerificationJobRepository.Update(job, tx)
	})
}

func (c *cHandlers) DirectoryCreate(cmd *stoservertypes.DirectoryCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.DirectoryRepository.Update(
			stotypes.NewDirectory(
				stoutils.NewDirectoryId(),
				cmd.Parent,
				cmd.Name),
			tx)
	})
}

func (c *cHandlers) DirectoryDelete(cmd *stoservertypes.DirectoryDelete, ctx *command.Ctx) error {
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

func (c *cHandlers) DirectoryRename(cmd *stoservertypes.DirectoryRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Name = cmd.Name

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeDescription(cmd *stoservertypes.DirectoryChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Description = cmd.Description

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeSensitivity(cmd *stoservertypes.DirectoryChangeSensitivity, ctx *command.Ctx) error {
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

func (c *cHandlers) DirectoryMove(cmd *stoservertypes.DirectoryMove, ctx *command.Ctx) error {
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

func (c *cHandlers) CollectionCreate(cmd *stoservertypes.CollectionCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		if _, err := stodb.Read(tx).Directory(cmd.ParentDir); err != nil {
			if err == blorm.ErrNotFound {
				return errors.New("parent directory not found")
			} else {
				return err
			}
		}

		// TODO: resolve this from closest parent that has policy defined?
		replicationPolicy, err := stodb.Read(tx).ReplicationPolicy("default")
		if err != nil {
			return err
		}

		encryptionKey := [32]byte{}
		if _, err := rand.Read(encryptionKey[:]); err != nil {
			return err
		}

		collection := &stotypes.Collection{
			ID:             stoutils.NewCollectionId(),
			Created:        time.Now(),
			Directory:      cmd.ParentDir,
			Name:           cmd.Name,
			DesiredVolumes: replicationPolicy.DesiredVolumes,
			Head:           stotypes.NoParentId,
			EncryptionKey:  encryptionKey,
			Changesets:     []stotypes.CollectionChangeset{},
			Metadata:       map[string]string{},
		}

		// highly unlikely
		if _, err := stodb.Read(tx).Collection(collection.ID); err != blorm.ErrNotFound {
			return errors.New("accidentally generated duplicate collection ID")
		}

		ctx.CreatedRecordId(collection.ID)

		return stodb.CollectionRepository.Update(collection, tx)
	})
}

func (c *cHandlers) CollectionChangeSensitivity(cmd *stoservertypes.CollectionChangeSensitivity, ctx *command.Ctx) error {
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

func (c *cHandlers) CollectionMove(cmd *stoservertypes.CollectionMove, ctx *command.Ctx) error {
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

func (c *cHandlers) CollectionChangeDescription(cmd *stoservertypes.CollectionChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Description = cmd.Description

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionRename(cmd *stoservertypes.CollectionRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Name = cmd.Name

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionFuseMount(cmd *stoservertypes.CollectionFuseMount, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	baseUrl, err := stodb.CfgFuseServerBaseUrl.GetRequired(tx)
	if err != nil {
		return err
	}

	vstofuse := stofuseclient.New(baseUrl)

	if cmd.UnmountOthers {
		if err := vstofuse.UnmountAll(); err != nil {
			return err
		}
	}

	return vstofuse.Mount(cmd.Collection)
}

func (c *cHandlers) CollectionDelete(cmd *stoservertypes.CollectionDelete, ctx *command.Ctx) error {
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

func (c *cHandlers) ClientCreate(cmd *stoservertypes.ClientCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.ClientRepository.Update(&stotypes.Client{
			ID:        stoutils.NewClientId(),
			Name:      cmd.Name,
			AuthToken: cryptorandombytes.Base64Url(32),
		}, tx)
	})
}

func (c *cHandlers) ClientRemove(cmd *stoservertypes.ClientRemove, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.ClientRepository.Delete(&stotypes.Client{
			ID: cmd.Id,
		}, tx)
	})
}

func (c *cHandlers) IntegrityverificationjobResume(cmd *stoservertypes.IntegrityverificationjobResume, ctx *command.Ctx) error {
	c.ivController.Resume(cmd.JobId)

	return nil
}

func (c *cHandlers) IntegrityverificationjobStop(cmd *stoservertypes.IntegrityverificationjobStop, ctx *command.Ctx) error {
	c.ivController.Stop(cmd.JobId)

	return nil
}

func (c *cHandlers) ReplicationpolicyChangeDesiredVolumes(cmd *stoservertypes.ReplicationpolicyChangeDesiredVolumes, ctx *command.Ctx) error {
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

func (c *cHandlers) ConfigSetFuseServerBaseurl(cmd *stoservertypes.ConfigSetFuseServerBaseurl, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return stodb.CfgFuseServerBaseUrl.Set(cmd.Baseurl, tx)
	})
}

func registerCommandEndpoints(
	router *mux.Router,
	eventLog eventlog.Log,
	cmdHandlers stoservertypes.CommandHandlers,
	mwares httpauth.MiddlewareChainMap,
) {
	router.HandleFunc("/command/{commandName}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commandName := mux.Vars(r)["commandName"]

		httpErr := httpcommand.Serve(
			w,
			r,
			mwares,
			commandName,
			stoservertypes.Allocators,
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
