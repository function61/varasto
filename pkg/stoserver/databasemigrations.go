package stoserver

import (
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"log"
	"time"
)

func (c *cHandlers) DatabaseMigrate(cmd *stoservertypes.DatabaseMigrate, ctx *command.Ctx) error {
	migrations := map[int]func(*bolt.Tx) error{
		1: migration1,
		2: migration2,
		3: migration3,
		4: migration4,
		5: migration5,
		6: migration6,
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		migration, found := migrations[cmd.Phase]
		if !found {
			return fmt.Errorf("migration not found for phase: %d", cmd.Phase)
		}

		return migration(tx)
	})
}

func migration1(tx *bolt.Tx) error {
	now := time.Now()

	if err := stodb.DirectoryRepository.Each(func(record interface{}) error {
		dir := record.(*stotypes.Directory)

		if dir.Type == "" {
			dir.Type = string(stoservertypes.DirectoryTypeGeneric)
			return stodb.DirectoryRepository.Update(dir, tx)
		}

		return nil
	}, tx); err != nil {
		return err
	}

	if err := stodb.VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)

		if vol.Technology == "" {
			vol.Technology = string(stoservertypes.VolumeTechnologyDiskHdd)
			return stodb.VolumeRepository.Update(vol, tx)
		}

		return nil
	}, tx); err != nil {
		return err
	}

	if err := stodb.ClientRepository.Each(func(record interface{}) error {
		client := record.(*stotypes.Client)

		if client.Created.IsZero() {
			client.Created = now
			return stodb.ClientRepository.Update(client, tx)
		}

		return nil
	}, tx); err != nil {
		return err
	}

	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		if coll.Tags == nil {
			coll.Tags = []string{}
			return stodb.CollectionRepository.Update(coll, tx)
		}

		return nil
	}, tx); err != nil {
		return err
	}

	return nil
}

func migration2(tx *bolt.Tx) error {
	return stodb.KeyEncryptionKeyRepository.Bootstrap(tx)
}

func migration3(tx *bolt.Tx) error {
	kenvs := map[string]stotypes.KeyEnvelope{}

	// 1st pass => collect all kenvs
	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		for _, kenv := range coll.EncryptionKeys {
			kenvs[kenv.KeyId] = kenv
		}

		return nil
	}, tx); err != nil {
		return err
	}

	// 2nd pass => scan for each collection's blobs with keyid not linked to
	// keyset of referencing collection
	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		thisCollectionsKenvs := map[string]bool{}

		for _, kenv := range coll.EncryptionKeys {
			thisCollectionsKenvs[kenv.KeyId] = true
		}

		if err := eachBlobOfCollection(coll, func(ref stotypes.BlobRef) error {
			blob, err := stodb.Read(tx).Blob(ref)
			if err != nil {
				return err
			}

			if _, has := thisCollectionsKenvs[blob.EncryptionKeyId]; !has {
				coll.EncryptionKeys = append(coll.EncryptionKeys, kenvs[blob.EncryptionKeyId])

				log.Printf("added key %s to coll %s", blob.EncryptionKeyId, coll.ID)

				thisCollectionsKenvs[blob.EncryptionKeyId] = true

				return stodb.CollectionRepository.Update(coll, tx)
			}

			return nil
		}); err != nil {
			return err
		}

		return nil
	}, tx); err != nil {
		return err
	}

	return nil
}

func migration4(tx *bolt.Tx) error {
	/*
		// collId => encryptionKeyId
		visitedCollections := map[string]string{}

		brokenCollections := map[string]bool{}

		kekPublicKeys := []rsa.PublicKey{}

		keks := []stotypes.KeyEncryptionKey{}
		if err := stodb.KeyEncryptionKeyRepository.Each(stodb.KeyEncryptionKeyAppender(&keks), tx); err != nil {
			return err
		}

		for _, kek := range keks {
			pubKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPublicKey(strings.NewReader(kek.PublicKey))
			if err != nil {
				return err
			}

			kekPublicKeys = append(kekPublicKeys, *pubKey)
		}

		if err := stodb.BlobRepository.Each(func(record interface{}) error {
			blob := record.(*stotypes.Blob)

			keyId, collVisited := visitedCollections[blob.Coll]
			if !collVisited {
				coll, err := stodb.Read(tx).Collection(blob.Coll)
				if err != nil {
					if _, seen := brokenCollections[blob.Coll]; !seen {
						brokenCollections[blob.Coll] = true
						log.Printf("coll %s not found", blob.Coll)
					}

					return stodb.BlobRepository.Delete(blob, tx)
				}

				keyId = stoutils.NewEncryptionKeyId()

				kenv, err := stotypes.EncryptEnvelope(keyId, coll.EncryptionKey[:], kekPublicKeys)
				if err != nil {
					panic(err)
				}

				coll.EncryptionKeys = []stotypes.KeyEnvelope{*kenv}
				coll.EncryptionKey = [32]byte{}

				if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
					return err
				}

				visitedCollections[blob.Coll] = keyId
			}

			blob.EncryptionKeyId = keyId
			blob.Coll = ""

			return stodb.BlobRepository.Update(blob, tx)
		}, tx); err != nil {
			return err
		}
	*/

	return nil
}

func migration5(tx *bolt.Tx) error {
	kek := &stotypes.KeyEncryptionKey{}
	if err := stodb.KeyEncryptionKeyRepository.OpenByPrimaryKey([]byte("0EWz"), kek, tx); err != nil {
		return err
	}

	kek.Bits = 4096

	return stodb.KeyEncryptionKeyRepository.Update(kek, tx)
}

func migration6(tx *bolt.Tx) error {
	if err := stodb.ScheduledJobRepository.Bootstrap(tx); err != nil {
		return err
	}

	for _, job := range []stotypes.ScheduledJob{
		{
			ID:          "ocKgpTHU3Sk",
			Description: "SMART poller",
			Schedule:    "@every 5m",
			Kind:        stoservertypes.ScheduledJobKindSmartpoll,
			Enabled:     true,
		},
		{
			ID:          "h-cPYsYtFzM",
			Description: "Metadata backup",
			Schedule:    "@midnight",
			Kind:        stoservertypes.ScheduledJobKindMetadatabackup,
			Enabled:     true,
		},
	} {
		job := job // pin

		if err := stodb.ScheduledJobRepository.Update(&job, tx); err != nil {
			return err
		}
	}

	return nil
}
