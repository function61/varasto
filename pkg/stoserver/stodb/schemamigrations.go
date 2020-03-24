package stodb

import (
	"encoding/binary"
	"fmt"
	"github.com/function61/varasto/pkg/blorm"
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
*/

const (
	CurrentSchemaVersion = 2
)

var (
	metaBucketKey    = []byte("_meta")
	schemaVersionKey = []byte("schemaVersion")
)

// returns blorm.ErrBucketNotFound if bootstrap required
func ValidateSchemaVersion(tx *bbolt.Tx) error {
	metaBucket := tx.Bucket(metaBucketKey)
	if metaBucket == nil {
		return blorm.ErrBucketNotFound
	}

	schemaVersionInDb := binary.LittleEndian.Uint32(metaBucket.Get(schemaVersionKey))

	if schemaVersionInDb != CurrentSchemaVersion {
		// migrations currently not implemented
		return fmt.Errorf(
			"incorrect schema version in DB: %d (expecting %d)",
			schemaVersionInDb,
			CurrentSchemaVersion)
	}

	return nil
}

func writeSchemaVersion(tx *bbolt.Tx) error {
	metaBucket, err := tx.CreateBucketIfNotExists(metaBucketKey)
	if err != nil {
		return err
	}

	schemaVersionInDb := make([]byte, 4)
	binary.LittleEndian.PutUint32(schemaVersionInDb[:], CurrentSchemaVersion)

	return metaBucket.Put(schemaVersionKey, schemaVersionInDb)
}
