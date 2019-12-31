package stodb

import (
	"encoding/binary"
	"fmt"
	"github.com/function61/varasto/pkg/blorm"
	"go.etcd.io/bbolt"
)

const (
	CurrentSchemaVersion = 1
)

var (
	metaBucketKey    = []byte("_meta")
	schemaVersionKey = []byte("schemaVersion")
)

// returns blorm.ErrBucketNotFound if bootstrap required
func ValidateSchemaVersion(tx *bolt.Tx) error {
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

func writeSchemaVersion(tx *bolt.Tx) error {
	metaBucket, err := tx.CreateBucketIfNotExists(metaBucketKey)
	if err != nil {
		return err
	}

	schemaVersionInDb := make([]byte, 4)
	binary.LittleEndian.PutUint32(schemaVersionInDb[:], CurrentSchemaVersion)

	return metaBucket.Put(schemaVersionKey, schemaVersionInDb)
}
