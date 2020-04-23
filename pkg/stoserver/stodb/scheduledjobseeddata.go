package stodb

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
)

// shared here b/c most of the definitions need to be in two places:
// 1) bootstrapper (= starting from fresh slate)
// 2) schema migration (adding scheduled job to existing installation)

func scheduledJobSeedSmartPoller() *stotypes.ScheduledJob {
	return &stotypes.ScheduledJob{
		ID:          "ocKgpTHU3Sk",
		Description: "SMART poller",
		Schedule:    "@every 5m",
		Kind:        stoservertypes.ScheduledJobKindSmartpoll,
		Enabled:     true,
	}
}

func scheduledJobSeedMetadataBackup() *stotypes.ScheduledJob {
	return &stotypes.ScheduledJob{
		ID:          "h-cPYsYtFzM",
		Description: "Metadata backup",
		Schedule:    "@midnight",
		Kind:        stoservertypes.ScheduledJobKindMetadatabackup,
		Enabled:     true,
	}
}

func scheduledJobSeedVersionUpdateCheck() *stotypes.ScheduledJob {
	return &stotypes.ScheduledJob{
		ID:          "EQi_3OhROUs",
		Description: "Varasto software update check",
		Schedule:    "@midnight",
		Kind:        stoservertypes.ScheduledJobKindVersionupdatecheck,
		Enabled:     true,
	}
}
