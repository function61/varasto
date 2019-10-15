package smart

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

type backend func(device string) ([]byte, error)

func Scan(device string, back backend) (*SmartCtlJsonReport, error) {
	smartCtlOutput, err := back(device)
	if err != nil {
		return nil, fmt.Errorf("%v, output: %s", err, smartCtlOutput)
	}

	return parseSmartCtlJsonReport(smartCtlOutput)
}

func SmartCtlBackend(device string) ([]byte, error) {
	stdout, err := exec.Command("smartctl", "--json", "--all", device).Output()

	return stdout, silenceSmartCtlAutomationHostileErrors(err)
}

/*	joonas/smartmontools built from simple Dockerfile:

	FROM alpine:edge
	RUN apk add --update smartmontools
*/
func SmartCtlDockerBackend(device string) ([]byte, error) {
	// disks in /dev are visible with --privileged but /dev/disk/by-uuid et al. are not
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

	return stdout, silenceSmartCtlAutomationHostileErrors(err)
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

func silenceSmartCtlAutomationHostileErrors(err error) error {
	if err != nil {
		if exitError, is := err.(*exec.ExitError); is {
			// unset bits 4-8 because they're not errors in getting the report itself
			// https://sourceforge.net/p/smartmontools/mailman/message/7330895/
			masked := exitError.ExitCode() &^ 0xf8

			if masked == 0 { // not error anymore
				return nil
			}
		}
	}

	return err
}
