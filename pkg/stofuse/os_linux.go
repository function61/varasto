package stofuse

// os-specific abstractions (Linux)

import (
	"syscall"
	"time"
)

func accessTimeFromStatt(stat *syscall.Stat_t, _ time.Time) time.Time {
	return timespecToTime(stat.Atim)
}
