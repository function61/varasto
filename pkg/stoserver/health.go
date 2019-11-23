package stoserver

import (
	"fmt"
	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"time"
)

func healthForScheduledJobs(tx *bolt.Tx) stohealth.HealthChecker {
	jobsHealth := []stohealth.HealthChecker{}

	jobs := []stotypes.ScheduledJob{}
	if err := stodb.ScheduledJobRepository.Each(stodb.ScheduledJobAppender(&jobs), tx); err != nil {
		panic(err)
	}

	now := time.Now()

	jobToHealth := func(job stotypes.ScheduledJob) stohealth.HealthChecker {
		if !job.Enabled {
			return stohealth.NewStaticHealthNode(
				job.Description,
				stoservertypes.HealthStatusWarn,
				"Job is disabled")
		}

		if job.LastRun == nil {
			return stohealth.NewStaticHealthNode(
				job.Description,
				stoservertypes.HealthStatusWarn,
				"Never run but enabled - wait for first execution")
		}

		if job.LastRun.Error != "" {
			return stohealth.NewStaticHealthNode(
				job.Description,
				stoservertypes.HealthStatusFail,
				"Last run failed - see scheduler for details")
		}

		return stohealth.NewStaticHealthNode(
			job.Description,
			stoservertypes.HealthStatusPass,
			fmt.Sprintf("OK %s ago", duration.Humanize(now.Sub(job.LastRun.Started))))
	}

	for _, job := range jobs {
		jobsHealth = append(jobsHealth, jobToHealth(job))
	}

	return stohealth.NewHealthFolder(
		"Scheduled jobs",
		jobsHealth...)
}

/*
	<=5 freezing (fail)
	5-45 => ok (pass)
	45-50 => uncomfortable  (warn)
	>=50 => too hot (fail)
*/
func temperatureToHealthStatus(tempC int) stoservertypes.HealthStatus {
	switch {
	case tempC <= 5: // freezing
		return stoservertypes.HealthStatusFail
	case tempC <= 45: // ok
		return stoservertypes.HealthStatusPass
	case tempC <= 50: // uncomfortable
		return stoservertypes.HealthStatusWarn
	default: // too hot
		return stoservertypes.HealthStatusFail
	}
}
