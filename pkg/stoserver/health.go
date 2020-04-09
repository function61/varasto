package stoserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
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
	quotasHealth := staticHealthBuilder("Quotas", stoservertypes.HealthKindVolumeMounts.Ptr())

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
	mountsHealth := staticHealthBuilder("Mounts online", stoservertypes.HealthKindVolumeMounts.Ptr())

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
			"OK %s ago (scan older than %s)",
			sinceHumanized,
			duration.Humanize(scanTooOldThreshold)))
	}

	return policyHealth.Pass(fmt.Sprintf("OK %s ago", sinceHumanized))
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
