package stohealth

import (
	"fmt"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"go.etcd.io/bbolt"
	"time"
)

var (
	cfgMetadatabackupLastSuccess = stodb.ConfigAccessor("metadatabackupLastSuccess")
)

func NewLastSuccessfullBackup(db *bolt.DB) HealthChecker {
	return &lastSuccessfullBackup{db}
}

type lastSuccessfullBackup struct {
	db *bolt.DB
}

func (h *lastSuccessfullBackup) CheckHealth() (*stoservertypes.Health, error) {
	tx, err := h.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	lastSuccessRaw, err := cfgMetadatabackupLastSuccess.GetOptional(tx)
	if err != nil {
		return nil, err
	}

	title := "Last metadata backup"

	if lastSuccessRaw == "" {
		return mkHealth(title, stoservertypes.HealthStatusFail, "Never taken, see: https://github.com/function61/varasto/blob/master/docs/guide_setting-up-backup.md")
	}

	lastSuccess, err := time.Parse(time.RFC3339, lastSuccessRaw)
	if err != nil {
		return nil, err
	}

	sinceLastSuccess := time.Since(lastSuccess)

	sinceLastSuccessHumanReadable := fmt.Sprintf("%d hour(s) since last backup", int(sinceLastSuccess.Hours()))

	if sinceLastSuccess > 48*time.Hour {
		return mkHealth(title, stoservertypes.HealthStatusFail, sinceLastSuccessHumanReadable)
	}

	if sinceLastSuccess > 24*time.Hour {
		return mkHealth(title, stoservertypes.HealthStatusWarn, sinceLastSuccessHumanReadable)
	}

	return mkHealth(title, stoservertypes.HealthStatusPass, sinceLastSuccessHumanReadable)
}

func UpdateMetadatabackupLastSuccess(timestamp time.Time, tx *bolt.Tx) error {
	return cfgMetadatabackupLastSuccess.Set(timestamp.Format(time.RFC3339), tx)
}