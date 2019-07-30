package stodb

import (
	"fmt"
	"github.com/function61/varasto/pkg/blorm"
	"go.etcd.io/bbolt"
)

type configAccessor struct {
	key []byte
}

func ConfigAccessor(key string) *configAccessor {
	return &configAccessor{[]byte(key)}
}

// returns blorm.ErrNotFound if bootstrap required
func (c *configAccessor) GetOptional(tx *bolt.Tx) (string, error) {
	return c.getWithRequired(false, tx)
}

// returns blorm.ErrNotFound if bootstrap required
// returns descriptive error message if value not set
func (c *configAccessor) GetRequired(tx *bolt.Tx) (string, error) {
	return c.getWithRequired(true, tx)
}

func (c *configAccessor) getWithRequired(required bool, tx *bolt.Tx) (string, error) {
	configBucket := tx.Bucket(configBucketKey)
	if configBucket == nil {
		return "", blorm.ErrNotFound
	}

	val := string(configBucket.Get(c.key))
	if val == "" && required {
		return "", fmt.Errorf("config value %s not set", c.key)
	}

	return val, nil
}

func (c *configAccessor) Set(value string, tx *bolt.Tx) error {
	configBucket := tx.Bucket(configBucketKey)
	if configBucket == nil {
		return blorm.ErrNotFound
	}

	return configBucket.Put(c.key, []byte(value))
}
