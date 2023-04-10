package stofuse

// os-specific abstractions (Darwin)

import (
	"syscall"
	"time"
)

// darwin doesn't seem to have field `Atim` in `syscall.Stat_t`
func accessTimeFromStatt(_ *syscall.Stat_t, modTime time.Time) time.Time {
	return modTime
}
