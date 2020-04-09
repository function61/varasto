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

	v3
	==
	  Changes: revamped replication policy infrastructure
	Migration: automatic
*/

const (
	CurrentSchemaVersion = 3
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

	schemaVersionInDb := binary.LittleEndian.Uint32(metaBucket.Get(schemaVersionKey))

	if schemaVersionInDb == CurrentSchemaVersion { // happy path => no migration needed
		return nil
	}

	// only support 2->3 for now
	if schemaVersionInDb != 2 {
		return fmt.Errorf(
			"schema migration %d -> %d not supported",
			schemaVersionInDb,
			CurrentSchemaVersion)
	}

	logex.Levels(logger).Info.Printf(
		"migrating from %d -> %d",
		schemaVersionInDb,
		CurrentSchemaVersion)

	if err := from2to3(tx); err != nil {
		return err
	}

	return writeSchemaVersionWith(3, tx)
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
