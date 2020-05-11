package stoserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"go.etcd.io/bbolt"
)

// in this file:
// - health checker (check if we're running the latest version)
// - command for getting the latest version from the update server
// - deserializer for the update JSON
// - scheduled job implementation for running the above command

func healthRunningLatestVersion(tx *bbolt.Tx) stohealth.HealthChecker {
	verCheck := staticHealthBuilder("Software updates", stoservertypes.HealthKindSoftwareUpdates.Ptr())

	checkResponse, err := stodb.CfgUpdateStatusAt.GetOptional(tx)
	if err != nil {
		panic(err)
	}

	if checkResponse == "" {
		return verCheck.Warn("Never checked (if you recently installed Varasto, wait 24h)")
	}

	statusAt, err := deserializeUpdateStatusAt(checkResponse)
	if err != nil {
		panic(err)
	}

	status := statusAt.Status // shorthand

	if dynversion.IsDevVersion() {
		return verCheck.Warn("Running dev version, therefore can´t judge if updates available")
	}

	// our version numbers are comparable. there could be published releases the users
	// are running that we haven't pushed as a stable release yet
	if dynversion.Version >= status.LatestVersion {
		return verCheck.Pass(
			fmt.Sprintf("Running latest version (checked %s ago)",
				duration.Humanize(time.Since(statusAt.At))))
	}

	return verCheck.Warn("Update available: " + status.LatestVersion)
}

func (c *cHandlers) NodeCheckForUpdates(cmd *stoservertypes.NodeCheckForUpdates, ctx *command.Ctx) error {
	// see https://function61.com/varasto/docs/security/privacy/
	endpoint := fmt.Sprintf(
		"https://function61.com/varasto/updateserver/latest-version.json?os=%s&arch=%s&version=%s",
		runtime.GOOS,
		runtime.GOARCH,
		dynversion.Version)

	status := stoservertypes.UpdatesStatus{}
	// intentionally allowing unknown fields to be forward-compatible if server adds data
	// (it won't be persisted though)
	res, err := ezhttp.Get(
		ctx.Ctx,
		endpoint,
		ezhttp.RespondsJson(&status, true))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// wrap status with StatusAt to record check timestamp.
	// need to marshal this wrapped thing back to JSON so we can persist it.
	statusAtJson, err := json.Marshal(&stoservertypes.UpdatesStatusAt{
		At:     time.Now().UTC(),
		Status: status,
	})
	if err != nil {
		return err
	}

	// check for validity before persisting
	if _, err := deserializeUpdateStatusAt(string(statusAtJson)); err != nil {
		return err
	}

	return c.setConfigValue(stodb.CfgUpdateStatusAt, string(statusAtJson))
}

// scheduled job for checking version updates
type versionUpdateCheckScheduledJob struct {
	commandPlumbing *scheduledJobCommandPlumbing
}

func (s *versionUpdateCheckScheduledJob) GetRunner() scheduler.JobFn {
	return commandInvokerJobFn(&stoservertypes.NodeCheckForUpdates{}, s.commandPlumbing)
}

func deserializeUpdateStatusAt(serialized string) (*stoservertypes.UpdatesStatusAt, error) {
	statusAt := &stoservertypes.UpdatesStatusAt{}
	if err := json.Unmarshal([]byte(serialized), statusAt); err != nil {
		return nil, fmt.Errorf("deserializeUpdateStatusAt: %w", err)
	}

	if statusAt.Status.LatestVersion == "" {
		return nil, errors.New("deserializeUpdateStatusAt: LatestVersion empty")
	}

	return statusAt, nil
}
