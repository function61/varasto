package stodb

import (
	"encoding/binary"

	"github.com/function61/varasto/pkg/blorm"
	"go.etcd.io/bbolt"
)

const (
	CurrentSchemaVersion = 6
)

var (
	metaBucketKey    = []byte("_meta")
	schemaVersionKey = []byte("schemaVersion")
)

// returns blorm.ErrBucketNotFound if version not found
func ReadSchemaVersion(tx *bbolt.Tx) (uint32, error) {
	metaBucket := tx.Bucket(metaBucketKey)
	if metaBucket == nil {
		return 0, blorm.ErrBucketNotFound
	}

	schemaVersionInDB := binary.LittleEndian.Uint32(metaBucket.Get(schemaVersionKey))
	return schemaVersionInDB, nil
}

func WriteSchemaVersion(version uint32, tx *bbolt.Tx) error {
	metaBucket, err := tx.CreateBucketIfNotExists(metaBucketKey)
	if err != nil {
		return err
	}

	schemaVersionInDB := make([]byte, 4)
	binary.LittleEndian.PutUint32(schemaVersionInDB, version)

	return metaBucket.Put(schemaVersionKey, schemaVersionInDB)
}

func writeSchemaVersionCurrent(tx *bbolt.Tx) error {
	return WriteSchemaVersion(CurrentSchemaVersion, tx)
}
