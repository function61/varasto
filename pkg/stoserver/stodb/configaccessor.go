package stodb

import (
	"fmt"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

type configAccessor struct {
	key string
}

func ConfigAccessor(key string) *configAccessor {
	return &configAccessor{key}
}

func (c *configAccessor) GetOptional(tx *bolt.Tx) (string, error) {
	return c.getWithRequired(false, tx)
}

// returns descriptive error message if value not set
func (c *configAccessor) GetRequired(tx *bolt.Tx) (string, error) {
	return c.getWithRequired(true, tx)
}

func (c *configAccessor) getWithRequired(required bool, tx *bolt.Tx) (string, error) {
	conf := &stotypes.Config{}
	if err := configRepository.OpenByPrimaryKey([]byte(c.key), conf, tx); err != nil && err != blorm.ErrNotFound {
		return "", err
	}

	if conf.Value == "" && required {
		return "", fmt.Errorf("config value %s not set", c.key)
	}

	return conf.Value, nil
}

func (c *configAccessor) Set(value string, tx *bolt.Tx) error {
	return configRepository.Update(&stotypes.Config{
		Key:   c.key,
		Value: value,
	}, tx)
}
