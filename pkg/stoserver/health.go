package stoserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/storeplication"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func healthForScheduledJobs(tx *bbolt.Tx) stohealth.HealthChecker {
	jobsHealth := []stohealth.HealthChecker{}

	jobs := []stotypes.ScheduledJob{}
	if err := stodb.ScheduledJobRepository.Each(stodb.ScheduledJobAppender(&jobs), tx); err != nil {
		panic(err)
	}

	now := time.Now()

	jobToHealth := func(job stotypes.ScheduledJob) stohealth.HealthChecker {
		jobHealth := staticHealthBuilder(job.Description, nil)

		if !job.Enabled {
			return jobHealth.Warn("Job is disabled")
		}

		if job.LastRun == nil {
			return jobHealth.Warn("Never run but enabled - wait for first execution")
		}

		if job.LastRun.Error != "" {
			return jobHealth.Fail("Last run failed - see scheduler for details")
		}

		return jobHealth.Pass(fmt.Sprintf(
			"OK %s ago",
			duration.Humanize(now.Sub(job.LastRun.Started))))
	}

	for _, job := range jobs {
		jobsHealth = append(jobsHealth, jobToHealth(job))
	}

	return stohealth.NewHealthFolder(
		"Scheduled jobs",
		stoservertypes.HealthKindScheduledJobs.Ptr(),
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

func quotaHealth(volumesOverQuota []string) stohealth.HealthChecker {
	quotasHealth := staticHealthBuilder("Quotas", stoservertypes.HealthKindVolume.Ptr())

	if len(volumesOverQuota) == 0 {
		return quotasHealth.Pass("")
	}

	return quotasHealth.Fail("Volumes over quota: " + strings.Join(volumesOverQuota, ", "))
}

func serverCertHealth(
	notAfter time.Time,
	now time.Time,
) stohealth.HealthChecker {
	timeLeft := notAfter.Sub(now)

	day := 24 * time.Hour // naive day

	status := func() stoservertypes.HealthStatus {
		switch {
		case timeLeft < 7*day:
			return stoservertypes.HealthStatusFail
		case timeLeft < 30*day:
			return stoservertypes.HealthStatusWarn
		default:
			return stoservertypes.HealthStatusPass
		}
	}()

	return stohealth.NewStaticHealthNode(
		"TLS certificate",
		status,
		fmt.Sprintf("Valid for %s", duration.Humanize(timeLeft)),
		stoservertypes.HealthKindTlsCertificate.Ptr())
}

func healthNoFailedMounts(failedMountNames []string) stohealth.HealthChecker {
	mountsHealth := staticHealthBuilder("Mounts online", stoservertypes.HealthKindMount.Ptr())

	if len(failedMountNames) > 0 {
		return mountsHealth.Fail(fmt.Sprintf(
			"Volumes errored: %s",
			strings.Join(failedMountNames, ", ")))
	}

	return mountsHealth.Pass("")
}

// - not scanned since Varasto restarted => warn
// - conflicts with policy => fail
// - scan older than 48 hours => warn
func healthNoReconciliationConflicts() stohealth.HealthChecker {
	policyHealth := staticHealthBuilder(
		"Replication policy scan",
		stoservertypes.HealthKindReplicationPolicies.Ptr())

	if latestReconciliationReport == nil {
		return policyHealth.Warn("Not ran since Varasto last started")
	}

	if len(latestReconciliationReport.CollectionsWithNonCompliantPolicy) > 0 {
		return policyHealth.Fail(fmt.Sprintf(
			"%d collection(s) conflict with its replication policy",
			len(latestReconciliationReport.CollectionsWithNonCompliantPolicy)))
	}

	since := time.Since(latestReconciliationReport.Timestamp)
	sinceHumanized := duration.Humanize(since)
	scanTooOldThreshold := 48 * time.Hour

	if since > scanTooOldThreshold {
		return policyHealth.Warn(fmt.Sprintf(
			"Last checked %s ago (scan older than %s)",
			sinceHumanized,
			duration.Humanize(scanTooOldThreshold)))
	}

	// warn about empty collections and directories (in case it's an accident and user
	// forgot to upload content)

	if len(latestReconciliationReport.EmptyCollectionIds) > 0 {
		return policyHealth.Warn(fmt.Sprintf("Empty collections: %s", strings.Join(latestReconciliationReport.EmptyCollectionIds, ", ")))
	}

	if len(latestReconciliationReport.EmptyDirectoryIds) > 0 {
		return policyHealth.Warn(fmt.Sprintf("Empty directories: %s", strings.Join(latestReconciliationReport.EmptyDirectoryIds, ", ")))
	}

	return policyHealth.Pass(fmt.Sprintf("OK %s ago", sinceHumanized))
}

func healthSubsystems(subsystems ...*subsystem) stohealth.HealthChecker {
	subsysHealths := []stohealth.HealthChecker{}

	stabilityJudgingPeriod := 15 * time.Second

	for _, subsys := range subsystems {
		if !subsys.enabled { // skip those we don't even want running
			continue
		}

		status := subsys.controller.Status()

		subsysHealths = append(subsysHealths, func() stohealth.HealthChecker {
			subsysHealth := staticHealthBuilder(status.Description, nil)

			if !status.Alive {
				return subsysHealth.Fail("Dead")
			}

			if time.Since(status.Started) < stabilityJudgingPeriod {
				return subsysHealth.Warn(fmt.Sprintf(
					"Started within %s - waiting to judge stability",
					stabilityJudgingPeriod))
			}

			return subsysHealth.Pass("Running stable")
		}())
	}

	return stohealth.NewHealthFolder(
		"Subsystems",
		stoservertypes.HealthKindSubsystems.Ptr(),
		subsysHealths...)
}

func healthVolReplication(
	vol *stotypes.Volume,
	tx *bbolt.Tx,
	conf *ServerConfig,
) (stohealth.HealthChecker, error) {
	volReplHealth := staticHealthBuilder(vol.Label, nil)

	replicationController, hasReplicationController := conf.ReplicationControllers[vol.ID]
	if hasReplicationController {
		replicationProgress := replicationController.Progress()

		if replicationProgress != 100 {
			return volReplHealth.Warn(fmt.Sprintf("Progress at %d %%", replicationProgress)), nil
		} else {
			return volReplHealth.Pass("Realtime"), nil
		}
	} else {
		anyQueued, err := storeplication.HasQueuedWriteIOsForVolume(vol.ID, tx)
		if err != nil {
			return nil, err
		}

		if anyQueued {
			return volReplHealth.Fail("Queued I/Os but replication paused"), nil
		} else {
			return nil, nil
		}
	}
}

type staticHealthFactory struct {
	name string
	kind *stoservertypes.HealthKind
}

func staticHealthBuilder(name string, kind *stoservertypes.HealthKind) *staticHealthFactory {
	return &staticHealthFactory{name, kind}
}

func (s *staticHealthFactory) Pass(details string) stohealth.HealthChecker {
	return stohealth.NewStaticHealthNode(s.name, stoservertypes.HealthStatusPass, details, s.kind)
}

func (s *staticHealthFactory) Warn(details string) stohealth.HealthChecker {
	return stohealth.NewStaticHealthNode(s.name, stoservertypes.HealthStatusWarn, details, s.kind)
}

func (s *staticHealthFactory) Fail(details string) stohealth.HealthChecker {
	return stohealth.NewStaticHealthNode(s.name, stoservertypes.HealthStatusFail, details, s.kind)
}
