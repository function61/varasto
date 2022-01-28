//go:build windows

package fssnapshot

import (
	"log"
)

func PlatformSpecificSnapshotter(logger *log.Logger) Snapshotter {
	return WindowsSnapshotter(logger)
}
