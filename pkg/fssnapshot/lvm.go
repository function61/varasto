//go:build linux

// must exclude from Windows build due to syscall.Mount(), syscall.Unmount()

package fssnapshot

// snapshots on Linux using LVM

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/function61/gokit/logex"
	"github.com/prometheus/procfs"
)

func LvmSnapshotter(snapshotSize string, logger *log.Logger) Snapshotter {
	return &lvmSnapshotter{snapshotSize, logex.Levels(logex.NonNil(logger))}
}

type lvmSnapshotter struct {
	snapshotSize string
	log          *logex.Leveled
}

func (l *lvmSnapshotter) Snapshot(path string) (*Snapshot, error) {
	procSelf, err := procfs.Self()
	if err != nil {
		return nil, err
	}

	mounts, err := procSelf.MountStats()
	if err != nil {
		return nil, err
	}

	mountOfOrigin := mountForPath(path, mounts)
	if mountOfOrigin == nil {
		return nil, errors.New("unable to resolve mount for path")
	}

	snapshotId := randomSnapId()

	//nolint:gosec // ok
	lvcreateOutput, err := exec.Command(
		"lvcreate",
		"--snapshot",
		"--size", l.snapshotSize,
		"--name", snapshotId,
		mountOfOrigin.Device).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"lvcreate failed: %s, output: %s",
			err.Error(),
			lvcreateOutput)
	}

	// we don't know the *device name* of the snapshot before using this command
	lvsOutput, err := exec.Command(
		"lvs",
		"--noheadings",
		"--options", "lv_name,lv_path").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"lvs failed: %s, output: %s",
			err.Error(),
			lvsOutput)
	}

	snapshotDevicePath := devicePathFromLvsOutput(snapshotId, lvsOutput)
	if snapshotDevicePath == "" {
		return nil, errors.New("failed to resolve snapshot path from lvs output")
	}

	completedSuccesfully := false

	defer func() {
		if completedSuccesfully {
			return
		}

		l.log.Info.Printf("cleaning up snapshot %s", snapshotId)

		if err := deleteLvmSnapshot(snapshotDevicePath); err != nil {
			l.log.Error.Printf("deleteLvmSnapshot: %v", err)
		}
	}()

	snapshotMountPath := filepath.Join("/mnt", snapshotId)

	if err := os.MkdirAll(snapshotMountPath, 0700); err != nil {
		return nil, fmt.Errorf(
			"failed to make directory %s for snapshot: %s",
			snapshotMountPath,
			err.Error())
	}

	defer func() {
		if completedSuccesfully {
			return
		}

		l.log.Info.Printf("cleanup: deleting mount path")

		if err := deleteLvmSnapshotMountPath(snapshotMountPath); err != nil {
			l.log.Error.Printf("deleteLvmSnapshotMountPath: %v", err)
		}
	}()

	if err := syscall.Mount(snapshotDevicePath, snapshotMountPath, mountOfOrigin.Type, 0, ""); err != nil {
		return nil, fmt.Errorf("mounting snapshot failed: %s", err.Error())
	}

	completedSuccesfully = true // cancel cleanups

	return &Snapshot{
		ID:                    snapshotDevicePath,
		OriginPath:            path,
		OriginInSnapshotPath:  originPathInSnapshot(path, mountOfOrigin.Mount, snapshotMountPath),
		SnapshotRootMountPath: snapshotMountPath,
	}, nil
}

func (l *lvmSnapshotter) Release(snapshot Snapshot) error {
	if err := syscall.Unmount(snapshot.SnapshotRootMountPath, 0); err != nil {
		return fmt.Errorf("unmounting snapshot failed: %s", err.Error())
	}

	if err := deleteLvmSnapshotMountPath(snapshot.SnapshotRootMountPath); err != nil {
		return err
	}

	return deleteLvmSnapshot(snapshot.ID)
}

func deleteLvmSnapshotMountPath(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove snapshotMountPath: %s", err.Error())
	}

	return nil
}

func deleteLvmSnapshot(snapshotPath string) error {
	removeOutput, err := exec.Command("lvremove", "--force", snapshotPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"lvremove for %s failed: %s, output: %s",
			snapshotPath,
			err.Error(),
			removeOutput)
	}

	return nil
}

func mountForPath(path string, mounts []*procfs.Mount) *procfs.Mount {
	var longestMatchingMount *procfs.Mount = nil

	for _, mount := range mounts {
		if !strings.HasPrefix(path, mount.Mount) || (longestMatchingMount != nil && len(mount.Mount) <= len(longestMatchingMount.Mount)) {
			continue
		}

		longestMatchingMount = mount
	}

	return longestMatchingMount
}

// see test for output example
var devicePathFromLvsOutputRe = regexp.MustCompile("^  ([^ ]+) +(.+)")

func devicePathFromLvsOutput(name string, output []byte) string {
	scanner := bufio.NewScanner(bytes.NewBuffer(output))
	for scanner.Scan() {
		matches := devicePathFromLvsOutputRe.FindStringSubmatch(scanner.Text())
		if matches == nil {
			continue
		}

		if matches[1] == name {
			return matches[2]
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}
