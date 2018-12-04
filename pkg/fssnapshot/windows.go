package fssnapshot

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

	snapshotId := findSnapshotIdFromCreateOutput(string(createSnapshotOutput))
	if snapshotId == "" {
		return nil, fmt.Errorf("unable to find snapshot ID from create output")
	}

	defer func() {
		if completedSuccesfully {
			return
		}

		w.log.Info.Printf("cleaning snapshot %s", snapshotId)

		if err := deleteSnapshot(snapshotId); err != nil {
			w.log.Error.Printf("cleaning up snapshot: %v", err)
		}
	}()

	getSnapshotDetailsOutput, err := exec.Command(
		"vssadmin",
		"list",
		"shadows",
		"/Shadow="+snapshotId).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"unable to list snapshot details: %s, output: %s",
			err.Error(),
			getSnapshotDetailsOutput)
	}

	snapshotDeviceId := findSnapshotDeviceFromDetailsOutput(string(getSnapshotDetailsOutput))
	if snapshotDeviceId == "" {
		return nil, fmt.Errorf("unable to find device ID from list output")
	}

	snapshotRootMountPath := driveLetter + ":/snapshots/" + randomSnapId()

	if err := os.MkdirAll(filepath.Dir(snapshotRootMountPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to make parent dir for snapshot mount: %s", err.Error())
	}

	// Windows makes a distinction between file and directory symlinks. os.Symlink()
	// doesn't seem to support directory type links on Windows. additionally, "mklink" is
	// a cmd-builtin, so we must invoke cmd to run mklink. Windows + CLI = LOLOLOL.
	// https://twitter.com/joonas_fi/status/1067810155872563200
	mklinkCmd := exec.Command(
		"cmd",
		"/c",
		"mklink",
		"/D",
		windowsPath(snapshotRootMountPath),
		windowsPath(snapshotDeviceId+"/"))
	mklinkOutput, err := mklinkCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make directory symlink: %s, output: %s",
			err.Error(),
			mklinkOutput)
	}

	completedSuccesfully = true // cancel cleanups

	return &Snapshot{
		ID:                    snapshotId,
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

func deleteSnapshot(shadowId string) error {
	removeSnapshotCmd := exec.Command(
		"vssadmin",
		"delete",
		"shadows",
		"/Quiet",
		"/Shadow="+shadowId)

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
	return strings.Replace(in, "/", `\`, -1)
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

var findSnapshotIdFromCreateOutputRe = regexp.MustCompile(`ShadowID = "([^ "]+)"`)

func findSnapshotIdFromCreateOutput(output string) string {
	match := findSnapshotIdFromCreateOutputRe.FindStringSubmatch(output)
	if match == nil {
		return ""
	}

	return match[1]
}
