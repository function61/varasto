package stohealth

import (
	"fmt"
	"time"

	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
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

	status := func() stoservertypes.HealthStatus {
		day := 24 * time.Hour // naive

		switch {
		case since > 30*day:
			return stoservertypes.HealthStatusFail
		case since > 14*day:
			return stoservertypes.HealthStatusWarn
		default:
			return stoservertypes.HealthStatusPass
		}
	}()

	return NewStaticHealthNode(
		"File integrity verification",
		status,
		sinceHumanReadable(since),
		stoservertypes.HealthKindVolumeIntegrity.Ptr(),
	).CheckHealth()
}

func sinceHumanReadable(since time.Duration) string {
	year := float64(24 * 365) // [h], naive

	// reference was zero
	if since.Hours() > 100*year {
		return "Never checked"
	}

	return fmt.Sprintf("%s since last check", duration.Humanize(since))
}

func ignoreError(err error) {
	// no-op
}
