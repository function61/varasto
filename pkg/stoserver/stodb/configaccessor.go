package stodb

import (
	"fmt"

	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

type ConfigRequiredError struct {
	error
}

type ConfigAccessor struct {
	key string
}

func configAccessor(key string) *ConfigAccessor {
	return &ConfigAccessor{key}
}

func (c *ConfigAccessor) GetOptional(tx *bbolt.Tx) (string, error) {
	return c.getWithRequired(false, tx)
}

// returns descriptive error message if value not set
func (c *ConfigAccessor) GetRequired(tx *bbolt.Tx) (string, error) {
	return c.getWithRequired(true, tx)
}

func (c *ConfigAccessor) getWithRequired(required bool, tx *bbolt.Tx) (string, error) {
	conf := &stotypes.Config{}
	if err := configRepository.OpenByPrimaryKey([]byte(c.key), conf, tx); err != nil && err != blorm.ErrNotFound {
		return "", err
	}

	if conf.Value == "" && required {
		return "", &ConfigRequiredError{fmt.Errorf("config value %s not set", c.key)}
	}

	return conf.Value, nil
}

func (c *ConfigAccessor) Set(value string, tx *bbolt.Tx) error {
	return configRepository.Update(&stotypes.Config{
		Key:   c.key,
		Value: value,
	}, tx)
}
