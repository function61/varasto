// +build !windows

package fssnapshot

import (
	"log"
)

func PlatformSpecificSnapshotter(logger *log.Logger) Snapshotter {
	return LvmSnapshotter("1GB", logger)
}
