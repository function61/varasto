package stoserver

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/httpcommand"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/blobstore/googledriveblobstore"
	"github.com/function61/varasto/pkg/blobstore/s3blobstore"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/smart"
	"github.com/function61/varasto/pkg/stofuse/stofuseclient"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodiskaccess"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stokeystore"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/samber/lo"
	"go.etcd.io/bbolt"
)

type cHandlers struct {
	db             *bbolt.DB
	conf           *ServerConfig
	ivController   *stointegrityverifier.Controller
	logger         *log.Logger
	configReloader *configReloader
}

func (c *cHandlers) VolumeCreate(cmd *stoservertypes.VolumeCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		maxVolID := 0

		allVolumes := []stotypes.Volume{}
		if err := stodb.VolumeRepository.Each(stodb.VolumeAppender(&allVolumes), tx); err != nil {
			return err
		}

		for _, vol := range allVolumes {
			if vol.ID > maxVolID {
				maxVolID = vol.ID
			}
		}

		return stodb.VolumeRepository.Update(&stotypes.Volume{
			ID:         maxVolID + 1,
			UUID:       stoutils.NewVolumeUUID(),
			Label:      cmd.Name,
			Technology: string(stoservertypes.VolumeTechnologyDiskHdd),
			Quota:      mebibytesToBytes(cmd.Quota),
			Zone:       "Default",
		}, tx)
	})
}

func (c *cHandlers) SubsystemStart(cmd *stoservertypes.SubsystemStart, ctx *command.Ctx) error {
	subsys := c.getSubsystem(cmd.Id)
	if subsys == nil {
		panic("shouldnt happen")
	}

	if subsys.enabled {
		return fmt.Errorf("subsystem %s already enabled", cmd.Id)
	}
	subsys.enabled = !subsys.enabled

	subsys.controller.Start()
	return nil
}

func (c *cHandlers) SubsystemStop(cmd *stoservertypes.SubsystemStop, ctx *command.Ctx) error {
	subsys := c.getSubsystem(cmd.Id)
	if subsys == nil {
		panic("shouldnt happen")
	}

	if !subsys.enabled {
		return fmt.Errorf("subsystem %s already disabled", cmd.Id)
	}
	subsys.enabled = !subsys.enabled

	subsys.controller.Stop()
	return nil
}

func (c *cHandlers) VolumeChangeQuota(cmd *stoservertypes.VolumeChangeQuota, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Quota = mebibytesToBytes(cmd.Quota)

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeSetManufacturingDate(cmd *stoservertypes.VolumeSetManufacturingDate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Manufactured = cmd.ManufacturingDate.Time

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeSetWarrantyEndDate(cmd *stoservertypes.VolumeSetWarrantyEndDate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.WarrantyEnds = cmd.WarrantyEndDate.Time

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeSetSerialNumber(cmd *stoservertypes.VolumeSetSerialNumber, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.SerialNumber = cmd.SerialNumber

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeSetTechnology(cmd *stoservertypes.VolumeSetTechnology, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Technology = string(cmd.Technology)

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeSetTopology(cmd *stoservertypes.VolumeSetTopology, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		if cmd.Slot == 0 && cmd.Enclosure != "" {
			return errors.New("'Slot' cannot be 0 when enclosure is defined")
		}

		vol.Enclosure = cmd.Enclosure
		vol.EnclosureSlot = cmd.Slot

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeZone(cmd *stoservertypes.VolumeChangeZone, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Zone = cmd.Zone

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeRename(cmd *stoservertypes.VolumeRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Label = cmd.Name

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeDescription(cmd *stoservertypes.VolumeChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Description = cmd.Description

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeChangeNotes(cmd *stoservertypes.VolumeChangeNotes, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.Notes = cmd.Notes

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) VolumeMountLocal(cmd *stoservertypes.VolumeMountLocal, ctx *command.Ctx) error {
	return c.mountVolume(
		cmd.Id,
		stoservertypes.VolumeDriverKindLocalFs,
		cmd.Path,
		ctx)
}

func (c *cHandlers) VolumeMountGoogleDrive(cmd *stoservertypes.VolumeMountGoogleDrive, ctx *command.Ctx) error {
	oauth2Config := googledriveblobstore.Oauth2Config(cmd.ClientId, cmd.ClientSecret)

	if cmd.AuthCode == "" {
		authCodeURL := googledriveblobstore.Oauth2AuthCodeURL(oauth2Config)

		// this is very much a hack
		ctx.CreatedRecordId(authCodeURL)
		return nil
	}

	oauth2Token, err := oauth2Config.Exchange(ctx.Ctx, cmd.AuthCode)
	if err != nil {
		return err
	}

	configSerialized, err := (&googledriveblobstore.Config{
		VarastoDirectoryID: cmd.FolderId,
		ClientID:           cmd.ClientId,
		ClientSecret:       cmd.ClientSecret,
		Token:              oauth2Token,
	}).Serialize()
	if err != nil {
		return err
	}

	if err := c.mountVolume(
		cmd.Id,
		stoservertypes.VolumeDriverKindGoogledrive,
		configSerialized,
		ctx); err != nil {
		return err
	}

	// this is very much a hack
	ctx.CreatedRecordId("mounted-ok")
	return nil
}

func (c *cHandlers) VolumeMountS3(cmd *stoservertypes.VolumeMountS3, ctx *command.Ctx) error {
	return c.mountVolume(
		cmd.Id,
		stoservertypes.VolumeDriverKindAwsS3,
		(&s3blobstore.Config{
			Bucket:          cmd.Bucket,
			Prefix:          cmd.PathPrefix,
			AccessKeyID:     cmd.AccessKeyId,
			AccessKeySecret: cmd.AccessKeySecret,
			RegionID:        cmd.RegionId,
			Endpoint:        cmd.Endpoint,
		}).Serialize(),
		ctx)
}

func (c *cHandlers) mountVolume(
	volID int,
	driverKind stoservertypes.VolumeDriverKind,
	driverOpts string,
	ctx *command.Ctx,
) error {
	sameVolumeOnSameNode := func(a, b stotypes.VolumeMount) bool {
		return a.Volume == b.Volume && a.Node == b.Node
	}

	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(volID)
		if err != nil {
			return err
		}

		mountSpec := &stotypes.VolumeMount{
			ID:         stoutils.NewVolumeMountID(),
			Volume:     vol.ID,
			Node:       c.conf.SelfNodeID,
			Driver:     driverKind,
			DriverOpts: driverOpts,
		}

		allMounts := []stotypes.VolumeMount{}
		if err := stodb.VolumeMountRepository.Each(stodb.VolumeMountAppender(&allMounts), tx); err != nil {
			return err
		}

		for _, otherMount := range allMounts {
			if sameVolumeOnSameNode(*mountSpec, otherMount) {
				return fmt.Errorf("same volume is already mounted at specified node. mount id: %s", otherMount.ID)
			}
		}

		// try mounting the volume
		driver, err := getDriver(*vol, *mountSpec, logex.Discard)
		if err != nil {
			return err
		}

		if err := c.conf.DiskAccess.Mountable(ctx.Ctx, vol.ID, vol.UUID, driver); err != nil {
			if err != stodiskaccess.ErrVolumeDescriptorNotFound {
				return err
			}

			if vol.BlobCount != 0 {
				return fmt.Errorf(
					"volume descriptor not found which is unexpected since volume has %d blob(s)",
					vol.BlobCount)
			}

			logex.Levels(c.logger).Info.Printf("initializing volume %s", vol.UUID)

			if err := c.conf.DiskAccess.Initialize(ctx.Ctx, vol.UUID, driver); err != nil {
				return fmt.Errorf("volume initialization failed: %v", err)
			}

			if err := c.conf.DiskAccess.Mountable(ctx.Ctx, vol.ID, vol.UUID, driver); err != nil {
				return fmt.Errorf(
					"volume not mountable after successful initialization: %v",
					err)
			}
		}

		return stodb.VolumeMountRepository.Update(mountSpec, tx)
	}))
}

func (c *cHandlers) VolumeUnmount(cmd *stoservertypes.VolumeUnmount, ctx *command.Ctx) error {
	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		mount, err := stodb.Read(tx).VolumeMount(cmd.Id)
		if err != nil {
			return err
		}

		return stodb.VolumeMountRepository.Delete(mount, tx)
	}))
}

// "copy any blobs that were on this volume, to another volume"
func (c *cHandlers) VolumeMigrateData(cmd *stoservertypes.VolumeMigrateData, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		from, err := stodb.Read(tx).Volume(cmd.From)
		if err != nil {
			return err
		}

		to, err := stodb.Read(tx).Volume(cmd.To)
		if err != nil {
			return err
		}

		if from.ID == to.ID {
			return fmt.Errorf("from == to (%d)", from.ID)
		}

		return stodb.BlobRepository.Each(func(record any) error {
			blob := record.(*stotypes.Blob)

			if !slices.Contains(blob.Volumes, from.ID) { // doesn't fit our criteria
				return nil
			}

			if slices.Contains(blob.Volumes, to.ID) { // is already in target volume
				return nil
			}

			blob.VolumesPendingReplication = append(blob.VolumesPendingReplication, to.ID)

			return stodb.BlobRepository.Update(blob, tx)
		}, tx)
	})
}

func (c *cHandlers) VolumeVerifyIntegrity(cmd *stoservertypes.VolumeVerifyIntegrity, ctx *command.Ctx) error {
	jobID := stoutils.NewIntegrityVerificationJobID()

	if err := c.db.Update(func(tx *bbolt.Tx) error {
		job := &stotypes.IntegrityVerificationJob{
			ID:       jobID,
			Started:  ctx.Meta.Timestamp,
			VolumeID: cmd.Id,
		}

		return stodb.IntegrityVerificationJobRepository.Update(job, tx)
	}); err != nil {
		return err
	}

	if cmd.Start {
		c.ivController.Resume(jobID)
	}

	return nil
}

func (c *cHandlers) DirectoryCreate(cmd *stoservertypes.DirectoryCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		if err := validateUniqueNameWithinSiblings(cmd.Parent, cmd.Name, tx); err != nil {
			return err
		}

		return stodb.DirectoryRepository.Update(
			stotypes.NewDirectory(
				stoutils.NewDirectoryID(),
				cmd.Parent,
				cmd.Name,
				string(stoservertypes.DirectoryTypeGeneric)),
			tx)
	})
}

func (c *cHandlers) DirectoryDelete(cmd *stoservertypes.DirectoryDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		if dir.MetaCollection != "" {
			metaColl, err := stodb.Read(tx).Collection(dir.MetaCollection)
			if err != nil {
				return err
			}

			if err := stodb.CollectionRepository.Delete(metaColl, tx); err != nil {
				return err
			}
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
			return fmt.Errorf("cannot delete directory because it has %d collection(s)", len(collections))
		}

		if len(subDirs) > 0 {
			return fmt.Errorf("cannot delete directory because it has %d directory(s)", len(subDirs))
		}

		return stodb.DirectoryRepository.Delete(dir, tx)
	})
}

func (c *cHandlers) DirectoryRename(cmd *stoservertypes.DirectoryRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Name = cmd.Name

		if err := validateUniqueNameWithinSiblings(dir.Parent, dir.Name, tx); err != nil {
			return err
		}

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeDescription(cmd *stoservertypes.DirectoryChangeDescription, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		// saves dir if metaColl is created on-the-fly
		metaColl, err := metaCollForDir(dir, tx, c.conf.KeyStore)
		if err != nil {
			return err
		}

		metaColl.Description = cmd.Description

		return stodb.CollectionRepository.Update(metaColl, tx)
	})
}

func (c *cHandlers) DirectorySetType(cmd *stoservertypes.DirectorySetType, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Id)
		if err != nil {
			return err
		}

		dir.Type = string(cmd.Type)

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

func (c *cHandlers) DirectoryChangeSensitivity(cmd *stoservertypes.DirectoryChangeSensitivity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
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
	return c.db.Update(func(tx *bbolt.Tx) error {
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

		if err := validateUniqueNameWithinSiblings(dirToMove.Parent, dirToMove.Name, tx); err != nil {
			return err
		}

		return stodb.DirectoryRepository.Update(dirToMove, tx)
	})
}

func (c *cHandlers) CollectionCreate(cmd *stoservertypes.CollectionCreate, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		collection, err := createCollection(cmd.Name, cmd.ParentDir, c.conf.KeyStore, tx)
		if err != nil {
			return err
		}

		ctx.CreatedRecordId(collection.ID)

		return stodb.CollectionRepository.Update(collection, tx)
	})
}

// caller is responsible for saving
func createCollection(
	name string,
	parentDirID string,
	keyStore *stokeystore.Store,
	tx *bbolt.Tx,
) (*stotypes.Collection, error) {
	parentDir, err := stodb.Read(tx).Directory(parentDirID)
	if err != nil {
		if err == blorm.ErrNotFound {
			return nil, errors.New("parent directory not found")
		} else {
			return nil, err
		}
	}

	if err := validateUniqueNameWithinSiblings(parentDir.ID, name, tx); err != nil {
		return nil, err
	}

	dirWithReplicationPolicy := parentDir
	for dirWithReplicationPolicy.ReplicationPolicy == "" {
		// root at the very latest specifies one
		dirWithReplicationPolicy, err = stodb.Read(tx).Directory(dirWithReplicationPolicy.Parent)
		if err != nil {
			return nil, err
		}
	}

	replicationPolicy, err := stodb.Read(tx).ReplicationPolicy(dirWithReplicationPolicy.ReplicationPolicy)
	if err != nil {
		return nil, err
	}

	if len(replicationPolicy.DesiredVolumes) == 0 {
		return nil, fmt.Errorf(
			"ReplicationPolicy %s doesn't specify any volumes",
			replicationPolicy.Name)
	}

	kekPubKeyFingerprints := []string{}

	keks := []stotypes.KeyEncryptionKey{}
	if err := stodb.KeyEncryptionKeyRepository.Each(stodb.KeyEncryptionKeyAppender(&keks), tx); err != nil {
		return nil, err
	}

	for _, kek := range keks {
		kekPubKeyFingerprints = append(kekPubKeyFingerprints, kek.Fingerprint)
	}

	dek := [32]byte{}
	if _, err := rand.Read(dek[:]); err != nil {
		return nil, err
	}

	// pack encryption key in an envelope protected with public key crypto,
	// so Varasto can store data without being able to access it itself
	dekEnvelopes, err := keyStore.EncryptDek(stoutils.NewEncryptionKeyID(), dek[:], kekPubKeyFingerprints)
	if err != nil {
		return nil, err
	}

	collection := &stotypes.Collection{
		ID:                stoutils.NewCollectionID(),
		Created:           time.Now(),
		Directory:         parentDir.ID,
		Name:              name,
		ReplicationPolicy: replicationPolicy.ID,
		Head:              stotypes.NoParentID,
		EncryptionKeys:    []stotypes.KeyEnvelope{*dekEnvelopes},
		Changesets:        []stotypes.CollectionChangeset{},
		Metadata:          map[string]string{},
		Tags:              []string{},
	}

	// highly unlikely
	if _, err := stodb.Read(tx).Collection(collection.ID); err != blorm.ErrNotFound {
		return nil, errors.New("accidentally generated duplicate collection ID")
	}

	collection.BumpGlobalVersion()

	return collection, nil
}

func (c *cHandlers) CollectionChangeSensitivity(cmd *stoservertypes.CollectionChangeSensitivity, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
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
	collIDs := *cmd.Collections
	if len(collIDs) == 0 {
		return nil // no-op
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		// check for existence
		if _, err := stodb.Read(tx).Directory(cmd.Directory); err != nil {
			return err
		}

		for _, collID := range collIDs {
			coll, err := stodb.Read(tx).Collection(collID)
			if err != nil {
				return err
			}

			if err := validateUniqueNameWithinSiblings(cmd.Directory, coll.Name, tx); err != nil {
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
	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Description = cmd.Description

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionRename(cmd *stoservertypes.CollectionRename, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		coll.Name = cmd.Name

		if err := validateUniqueNameWithinSiblings(coll.Directory, coll.Name, tx); err != nil {
			return err
		}

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionRate(cmd *stoservertypes.CollectionRate, ctx *command.Ctx) error {
	if cmd.Rating < 1 || cmd.Rating > 5 {
		return fmt.Errorf("'Rating' must be 1-5; got: %d", cmd.Rating)
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		if coll.Rating == cmd.Rating {
			return errors.New("rating unchanged")
		}

		coll.Rating = cmd.Rating

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionTag(cmd *stoservertypes.CollectionTag, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Id)
		if err != nil {
			return err
		}

		if slices.Contains(coll.Tags, cmd.Tag) {
			return fmt.Errorf("already tagged: %s", cmd.Tag)
		}

		coll.Tags = append(coll.Tags, cmd.Tag)

		sort.Strings(coll.Tags)

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) CollectionUntag(cmd *stoservertypes.CollectionUntag, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Id)
		if err != nil {
			return err
		}

		if !slices.Contains(coll.Tags, cmd.Tag) {
			return fmt.Errorf("not tagged: %s", cmd.Tag)
		}

		coll.Tags = lo.Filter(coll.Tags, func(tag string, _ int) bool { return tag != cmd.Tag })

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) FuseUnmountAll(cmd *stoservertypes.FuseUnmountAll, ctx *command.Ctx) error {
	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	baseURL, err := stodb.CfgFuseServerBaseURL.GetRequired(tx)
	if err != nil {
		return err
	}

	vstofuse := stofuseclient.New(baseURL)

	return vstofuse.UnmountAll(ctx.Ctx)
}

func (c *cHandlers) CollectionDelete(cmd *stoservertypes.CollectionDelete, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
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

func (c *cHandlers) ApikeyCreate(cmd *stoservertypes.ApikeyCreate, ctx *command.Ctx) error {
	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		return stodb.ClientRepository.Update(&stotypes.Client{
			ID:        stoutils.NewClientID(),
			Created:   ctx.Meta.Timestamp,
			Name:      cmd.Name,
			AuthToken: stoutils.NewAPIKeySecret(),
		}, tx)
	}))
}

func (c *cHandlers) ApikeyRemove(cmd *stoservertypes.ApikeyRemove, ctx *command.Ctx) error {
	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		return stodb.ClientRepository.Delete(&stotypes.Client{
			ID: cmd.Id,
		}, tx)
	}))
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
	return c.db.Update(func(tx *bbolt.Tx) error {
		desiredVolumes := []int{}

		for _, desiredVolumeID := range []int{cmd.Volume1, cmd.Volume2, cmd.Volume3, cmd.Volume4} {
			if desiredVolumeID == 0 { // null value for int
				continue
			}

			// verify that each volume exists
			if _, err := stodb.Read(tx).Volume(desiredVolumeID); err != nil {
				return fmt.Errorf("desiredVolume %d: %v", desiredVolumeID, err)
			}

			if slices.Contains(desiredVolumes, desiredVolumeID) {
				return fmt.Errorf("volume %d specified twice", desiredVolumeID)
			} else {
				desiredVolumes = append(desiredVolumes, desiredVolumeID)
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
	return c.setConfigValue(stodb.CfgFuseServerBaseURL, cmd.Baseurl)
}

func (c *cHandlers) ConfigSetGrafanaURL(cmd *stoservertypes.ConfigSetGrafanaURL, ctx *command.Ctx) error {
	return c.setConfigValue(stodb.CfgGrafanaURL, cmd.Url)
}

func (c *cHandlers) ConfigSetMediascannerState(cmd *stoservertypes.ConfigSetMediascannerState, ctx *command.Ctx) error {
	return c.setConfigValue(stodb.CfgMediascannerState, cmd.State)
}

func (c *cHandlers) ConfigSetNetworkShareBaseURL(cmd *stoservertypes.ConfigSetNetworkShareBaseURL, ctx *command.Ctx) error {
	return c.setConfigValue(stodb.CfgNetworkShareBaseURL, cmd.Baseurl)
}

func (c *cHandlers) setConfigValue(config *stodb.ConfigAccessor, newValue string) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		return config.Set(newValue, tx)
	})
}

func (c *cHandlers) VolumeSmartSetID(cmd *stoservertypes.VolumeSmartSetID, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		vol, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		vol.SmartID = cmd.SmartId

		return stodb.VolumeRepository.Update(vol, tx)
	})
}

func (c *cHandlers) getSubsystem(id stoservertypes.SubsystemId) *subsystem {
	switch stoservertypes.SubsystemIdExhaustive057882(id) {
	case stoservertypes.SubsystemIdMediascanner:
		return c.conf.MediaScanner
	case stoservertypes.SubsystemIdFuseProjector:
		return c.conf.FuseProjector
	default:
		return nil
	}
}

func (c *cHandlers) NodeInstallTLSCert(cmd *stoservertypes.NodeInstallTLSCert, ctx *command.Ctx) error {
	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		node, err := stodb.Read(tx).Node(cmd.Id)
		if err != nil {
			return err
		}

		node.TLSCert = cmd.TlsCertificate

		// changing private key? (does not necessarily change when cert is renewed)
		if cmd.TlsCertificatePrivateKey != "" {
			if err := stodb.CfgNodeTLSCertKey.Set(cmd.TlsCertificatePrivateKey, tx); err != nil {
				return err
			}
		}

		privKeyPem, err := stodb.CfgNodeTLSCertKey.GetRequired(tx)
		if err != nil {
			return err
		}

		// validate that cert & private key:
		//   1) parse
		//   2) match each other
		if _, err := tls.X509KeyPair([]byte(node.TLSCert), []byte(privKeyPem)); err != nil {
			return err
		}

		return stodb.NodeRepository.Update(node, tx)
	}))
}

func (c *cHandlers) NodeChangeSmartBackend(cmd *stoservertypes.NodeChangeSmartBackend, ctx *command.Ctx) error {
	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		node, err := stodb.Read(tx).Node(cmd.Id)
		if err != nil {
			return err
		}

		node.SmartBackend = cmd.Backend

		return stodb.NodeRepository.Update(node, tx)
	}))
}

func (c *cHandlers) NodeSmartScan(cmd *stoservertypes.NodeSmartScan, ctx *command.Ctx) error {
	type smartCapableVolume struct {
		volID   int
		smartID string
		report  *stoservertypes.SmartReport
	}

	scans := []*smartCapableVolume{}

	// list volumes that are capable of their SMART scan (for example cloud volumes obviously are not)
	if err := c.db.View(func(tx *bbolt.Tx) error {
		return stodb.VolumeRepository.Each(func(record any) error {
			vol := record.(*stotypes.Volume)

			// skip volume if SMART collection is not enabled for it (OR it's not mounted)
			if vol.SmartID == "" || !c.conf.DiskAccess.IsMounted(vol.ID) {
				return nil
			}

			scans = append(scans, &smartCapableVolume{
				volID:   vol.ID,
				smartID: vol.SmartID,
			})

			return nil
		}, tx)
	}); err != nil {
		return err
	}

	smartBackend, err := getSmartBackend(c.conf.SelfNodeSmartBackend)
	if err != nil {
		return fmt.Errorf("getSmartBackend: %v", err)
	}

	for _, scan := range scans {
		report, err := smart.Scan(scan.smartID, smartBackend)
		if err != nil {
			return fmt.Errorf("vol %d (%s) SMART: %v", scan.volID, scan.smartID, err)
		}

		var temp *int
		var powerOnTime *int
		var powerCycleCount *int

		if report.Temperature.Current != 0 {
			temp = &report.Temperature.Current
		}

		if report.PowerOnTime.Hours != 0 {
			powerOnTime = &report.PowerOnTime.Hours
		}

		if report.PowerCycleCount != 0 {
			powerCycleCount = &report.PowerCycleCount
		}

		scan.report = &stoservertypes.SmartReport{
			Time:            time.Now(),
			Passed:          report.SmartStatus.Passed,
			Temperature:     temp,
			PowerCycleCount: powerCycleCount,
			PowerOnTime:     powerOnTime,
		}
	}

	// nothing to do
	if len(scans) == 0 {
		return nil
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		for _, scan := range scans {
			vol, err := stodb.Read(tx).Volume(scan.volID)
			if err != nil {
				return err
			}

			// update back into db
			smartReportJSON, err := json.Marshal(scan.report)
			if err != nil {
				return err
			}

			vol.SmartReport = string(smartReportJSON)

			if err := stodb.VolumeRepository.Update(vol, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

// helper for reloading config if underlying operation succeeded
func (c *cHandlers) confreload(err error) error {
	if err == nil {
		logex.Levels(c.logger).Info.Println("reloading config in a few seconds")

		c.configReloader.ReloadConfig()
	}

	return err
}

func registerCommandEndpoints(
	router *http.ServeMux,
	eventLog eventlog.Log,
	invoker command.Invoker,
	mwares httpauth.MiddlewareChainMap,
) {
	router.HandleFunc("POST /command/{commandName}", func(w http.ResponseWriter, r *http.Request) {
		commandName := r.PathValue("commandName")

		httpErr := httpcommand.Serve(
			w,
			r,
			mwares,
			commandName,
			stoservertypes.Allocators,
			invoker,
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
			_, _ = w.Write([]byte(`{}`))
		}
	})
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

// conflict could arise, when directory OR collection:
// - is created as a sibling with non-unique name
// - is renamed to non-unique name
// - once unique-within-siblings item is moved into a directory where name already exists
func validateUniqueNameWithinSiblings(dirID string, name string, tx *bbolt.Tx) error {
	siblingDirectories, err := stodb.Read(tx).SubDirectories(dirID)
	if err != nil {
		return err
	}

	siblingCollections, err := stodb.Read(tx).CollectionsByDirectory(dirID)
	if err != nil {
		return err
	}

	for _, siblingDirectory := range siblingDirectories {
		if siblingDirectory.Name == name {
			return fmt.Errorf("directory %s already exists as a sibling", name)
		}
	}

	for _, siblingCollection := range siblingCollections {
		if siblingCollection.Name == name {
			return fmt.Errorf("collection %s already exists as a sibling", name)
		}
	}

	return nil
}

func getSmartBackend(typ stoservertypes.SmartBackend) (smart.Backend, error) {
	switch stoservertypes.SmartBackendExhaustive7712fd(typ) {
	case stoservertypes.SmartBackendSmartCtlViaDocker:
		return smart.SmartCtlViaDockerBackend, nil
	case stoservertypes.SmartBackendSmartCtl:
		return smart.SmartCtlBackend, nil
	default:
		return nil, fmt.Errorf("unsupported SmartBackend: %s", typ)
	}
}
