//go:build linux

package fssnapshot

import (
	"log"
)

func PlatformSpecificSnapshotter(logger *log.Logger) Snapshotter {
	return LvmSnapshotter("1GB", logger)
}
