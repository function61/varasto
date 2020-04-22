package stodb

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

/*	Schema versions

	v0
	==
	  Changes: no versioning in use
	Migration: n/a

	v1
	==
	  Changes: started schema versioning, added SMART backend type detection to bootstrap
	Migration: change backup header to signature vN, add your desired SMART backend to Node JSON

	v2
	==
	  Changes: added index to DEK to bring back deduplication
	Migration: change backup header to signature vN, import

	v3 (and onwards)
	==
	  Changes: described in function comment
	Migration: automatic
*/

const (
	CurrentSchemaVersion = 4
)

var (
	metaBucketKey    = []byte("_meta")
	schemaVersionKey = []byte("schemaVersion")
)

// returns blorm.ErrBucketNotFound if bootstrap required
func ValidateSchemaVersion(tx *bbolt.Tx, logger *log.Logger) error {
	metaBucket := tx.Bucket(metaBucketKey)
	if metaBucket == nil {
		return blorm.ErrBucketNotFound
	}

	// the migrations will continue until morale improves
	for {
		schemaVersionInDb := binary.LittleEndian.Uint32(metaBucket.Get(schemaVersionKey))

		// no migration needed (or migrations reached a happy level)
		if schemaVersionInDb == CurrentSchemaVersion {
			return nil
		}

		schemaVersionAfterMigration := schemaVersionInDb + 1

		logex.Levels(logger).Info.Printf(
			"migrating from %d -> %d",
			schemaVersionInDb,
			schemaVersionAfterMigration)

		if err := migrate(schemaVersionInDb, tx); err != nil {
			return err
		}

		if err := writeSchemaVersionWith(schemaVersionAfterMigration, tx); err != nil {
			return err
		}
	}
}

func migrate(schemaVersionInDb uint32, tx *bbolt.Tx) error {
	switch schemaVersionInDb {
	case 2:
		return from2to3(tx)
	case 3:
		return from3to4(tx)
	default:
		return fmt.Errorf(
			"schema migration %d -> %d not supported",
			schemaVersionInDb,
			schemaVersionInDb+1)
	}
}

// sets these attributes:
// - directory.ReplicationPolicy
// - coll.ReplicationPolicy
// - volume.Zone
// - replicationPolicy.Zones = 1
func from2to3(tx *bbolt.Tx) error {
	if err := CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)
		coll.ReplicationPolicy = "default"
		return CollectionRepository.Update(coll, tx)
	}, tx); err != nil {
		return err
	}

	dir, err := Read(tx).Directory(stoservertypes.RootFolderId)
	if err != nil {
		return err
	}
	dir.ReplicationPolicy = "default"
	if err := DirectoryRepository.Update(dir, tx); err != nil {
		return err
	}

	if err := VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)
		vol.Zone = "Default"
		return VolumeRepository.Update(vol, tx)
	}, tx); err != nil {
		return err
	}

	if err := ReplicationPolicyRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.ReplicationPolicy)
		vol.MinZones = 1
		return ReplicationPolicyRepository.Update(vol, tx)
	}, tx); err != nil {
		return err
	}

	return nil
}

// add scheduled task: "update check"
func from3to4(tx *bbolt.Tx) error {
	return ScheduledJobRepository.Update(scheduledJobSeedVersionUpdateCheck(), tx)
}

func writeSchemaVersion(tx *bbolt.Tx) error {
	return writeSchemaVersionWith(CurrentSchemaVersion, tx)
}

func writeSchemaVersionWith(version uint32, tx *bbolt.Tx) error {
	metaBucket, err := tx.CreateBucketIfNotExists(metaBucketKey)
	if err != nil {
		return err
	}

	schemaVersionInDb := make([]byte, 4)
	binary.LittleEndian.PutUint32(schemaVersionInDb[:], version)

	return metaBucket.Put(schemaVersionKey, schemaVersionInDb)
}
