package stohealth

import (
	"fmt"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"time"
)

func NewLastIntegrityVerificationJob(db *bbolt.DB) HealthChecker {
	return &lastIvJob{db}
}

type lastIvJob struct {
	db *bbolt.DB
}

// TODO: this check only checks the latest completed check, and trusts the user having
// ran it for each applicable volume
func (h *lastIvJob) CheckHealth() (*stoservertypes.Health, error) {
	tx, err := h.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	newest := time.Time{}

	if err := stodb.IntegrityVerificationJobRepository.Each(func(record interface{}) error {
		job := record.(*stotypes.IntegrityVerificationJob)

		if job.Completed.After(newest) {
			newest = job.Completed
		}

		return nil
	}, tx); err != nil {
		return nil, err
	}

	since := time.Since(newest)

	title := "File integrity verification"

	sinceHumanReadable := fmt.Sprintf("%d day(s) since last check", int(since.Hours()/24))

	if newest.IsZero() {
		sinceHumanReadable = "Never checked"
	}

	if since > naiveDays(30) {
		return mkHealth(title, stoservertypes.HealthStatusFail, sinceHumanReadable)
	}

	if since > naiveDays(14) {
		return mkHealth(title, stoservertypes.HealthStatusWarn, sinceHumanReadable)
	}

	return mkHealth(title, stoservertypes.HealthStatusPass, sinceHumanReadable)
}

func naiveDays(amount time.Duration) time.Duration {
	return amount * 24 * time.Hour
}

func ignoreError(err error) {
	// no-op
}
