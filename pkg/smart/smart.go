// Access SMART data of disks
package smart

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// output is JSON compatible with SmartCtlJsonReport
type Backend func(device string) ([]byte, error)

func Scan(device string, back Backend) (*SmartCtlJsonReport, error) {
	smartCtlOutput, err := back(device)
	if err != nil {
		return nil, fmt.Errorf("%v, output: %s", err, smartCtlOutput)
	}

	return parseSmartCtlJsonReport(smartCtlOutput)
}

func SmartCtlBackend(device string) ([]byte, error) {
	stdout, err := exec.Command("smartctl", "--json", "--all", device).Output()

	return stdout, SilenceSmartCtlAutomationHostileErrors(err)
}

/*	joonas/smartmontools built from simple Dockerfile:

	FROM alpine:edge
	RUN apk add --update smartmontools
*/
func SmartCtlViaDockerBackend(device string) ([]byte, error) {
	// disks in /dev are visible without --privileged (by mapping /dev:/dev) but
	// /dev/disk/by-uuid et al. are not
	// maybe related: https://github.com/moby/moby/issues/16160
	stdout, err := exec.Command(
		"docker", "run",
		"--rm",
		"-t",
		"--privileged",
		"-v", "/dev:/dev:ro",
		"joonas/smartmontools:20191015",
		"smartctl",
		"--json",
		"--all",
		device,
	).Output()

	return stdout, SilenceSmartCtlAutomationHostileErrors(err)
}

func parseSmartCtlJsonReport(reportJson []byte) (*SmartCtlJsonReport, error) {
	rep := &SmartCtlJsonReport{}

	if err := json.Unmarshal(reportJson, rep); err != nil {
		return nil, err
	}

	if len(rep.JsonFormatVersion) < 2 || rep.JsonFormatVersion[0] != 1 {
		return nil, errors.New("invalid json_format_version")
	}

	return rep, nil
}

// exported because used from outside
func SilenceSmartCtlAutomationHostileErrors(err error) error {
	if err != nil {
		if exitError, is := err.(*exec.ExitError); is {
			/* https://linux.die.net/man/8/smartctl documents $ smartctl return values:

			bits 0-1 as actually smartctl invocation errors
			bit 2 should be an error, but I got syntactically valid reports back with this up
			bits 3-7 are not smartctl invocation errors, but drive pre-fail or fail errors

			therefore we need to unset bits 2-7 (conservative unset would be 3-7)
			*/
			masked := exitError.ExitCode() &^ 0b11111100

			if masked == 0 { // not error anymore
				return nil
			}
		}
	}

	return err
}
