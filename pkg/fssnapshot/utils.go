package fssnapshot

import (
	"github.com/function61/gokit/cryptorandombytes"
	"path/filepath"
)

func randomSnapId() string {
	return "snap-" + cryptorandombytes.Hex(4)
}

// see tests for what this does
func originPathInSnapshot(originPath string, mountPoint string, snapshotPath string) string {
	return filepath.Join(
		snapshotPath,
		originPath[len(mountPoint):])
}
