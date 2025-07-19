package fssnapshot

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/function61/gokit/logex"
)

// I wrote an overview of this process @ https://github.com/restic/restic/issues/340#issuecomment-442446540
// thanks for pointers: https://github.com/restic/restic/issues/340#issuecomment-307636386

func WindowsSnapshotter(logger *log.Logger) Snapshotter {
	return &windowsSnapshotter{
		log: logex.Levels(logex.NonNil(logger)),
	}
}

type windowsSnapshotter struct {
	log *logex.Leveled
}

func (w *windowsSnapshotter) Snapshot(path string) (*Snapshot, error) {
	completedSuccesfully := false

	driveLetter := driveLetterFromPath(path)

	// Microsoft being the usual dick that M$FT is, they disable creating snapshots from
	// vssadmin on non-server OSs, therefore we must bypass the restriction by using wmic
	// instead. https://superuser.com/a/1125605/284803
	//
	//nolint:gosec // ok
	createSnapshotOutput, err := exec.Command(
		"wmic",
		"shadowcopy",
		"call",
		"create",
		fmt.Sprintf(`Volume="%s:\"`, driveLetter)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"error creating snapshot: %s, output: %s",
			err.Error(),
			createSnapshotOutput)
	}

	snapshotID := findSnapshotIDFromCreateOutput(string(createSnapshotOutput))
	if snapshotID == "" {
		return nil, fmt.Errorf("unable to find snapshot ID from create output")
	}

	defer func() {
		if completedSuccesfully {
			return
		}

		w.log.Info.Printf("cleaning snapshot %s", snapshotID)

		if err := deleteSnapshot(snapshotID); err != nil {
			w.log.Error.Printf("cleaning up snapshot: %v", err)
		}
	}()

	//nolint:gosec // ok
	getSnapshotDetailsOutput, err := exec.Command(
		"vssadmin",
		"list",
		"shadows",
		"/Shadow="+snapshotID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"unable to list snapshot details: %s, output: %s",
			err.Error(),
			getSnapshotDetailsOutput)
	}

	snapshotDeviceID := findSnapshotDeviceFromDetailsOutput(string(getSnapshotDetailsOutput))
	if snapshotDeviceID == "" {
		return nil, fmt.Errorf("unable to find device ID from list output")
	}

	snapshotRootMountPath := driveLetter + ":/snapshots/" + randomSnapID()

	if err := os.MkdirAll(filepath.Dir(snapshotRootMountPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to make parent dir for snapshot mount: %s", err.Error())
	}

	// Windows makes a distinction between file and directory symlinks. os.Symlink()
	// doesn't seem to support directory type links on Windows. additionally, "mklink" is
	// a cmd-builtin, so we must invoke cmd to run mklink. Windows + CLI = LOLOLOL.
	// https://twitter.com/joonas_fi/status/1067810155872563200
	//
	//nolint:gosec // ok
	mklinkCmd := exec.Command(
		"cmd",
		"/c",
		"mklink",
		"/D",
		windowsPath(snapshotRootMountPath),
		windowsPath(snapshotDeviceID+"/"))
	mklinkOutput, err := mklinkCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make directory symlink: %s, output: %s",
			err.Error(),
			mklinkOutput)
	}

	completedSuccesfully = true // cancel cleanups

	return &Snapshot{
		ID:                    snapshotID,
		OriginPath:            path,
		OriginInSnapshotPath:  originPathInSnapshot(path, driveLetter+":/", snapshotRootMountPath),
		SnapshotRootMountPath: snapshotRootMountPath,
	}, nil
}

func (w *windowsSnapshotter) Release(snap Snapshot) error {
	if err := deleteSnapshot(snap.ID); err != nil {
		return err
	}

	if err := os.Remove(snap.SnapshotRootMountPath); err != nil {
		return fmt.Errorf("unable to remove Snapshot SnapshotRootMountPath: %s", err.Error())
	}

	return nil
}

func deleteSnapshot(shadowID string) error {
	//nolint:gosec // ok
	removeSnapshotCmd := exec.Command(
		"vssadmin",
		"delete",
		"shadows",
		"/Quiet",
		"/Shadow="+shadowID)

	removeSnapshotOutput, err := removeSnapshotCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"unable to remove Snapshot: %s, output: %s",
			err.Error(),
			removeSnapshotOutput)
	}

	return nil
}

// '/' => '\'
// FIXME: maybe Go has more idiomatic way for this?
func windowsPath(in string) string {
	return strings.ReplaceAll(in, "/", `\`)
}

func driveLetterFromPath(path string) string {
	return path[0:1]
}

var findSnapshotDeviceFromDetailsOutputRe = regexp.MustCompile("Shadow Copy Volume: (.+)")

func findSnapshotDeviceFromDetailsOutput(output string) string {
	match := findSnapshotDeviceFromDetailsOutputRe.FindStringSubmatch(output)
	if match == nil {
		return ""
	}

	return match[1]
}

var findSnapshotIDFromCreateOutputRe = regexp.MustCompile(`ShadowID = "([^ "]+)"`)

func findSnapshotIDFromCreateOutput(output string) string {
	match := findSnapshotIDFromCreateOutputRe.FindStringSubmatch(output)
	if match == nil {
		return ""
	}

	return match[1]
}
