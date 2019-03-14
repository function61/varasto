package appuptime

import (
	"time"
)

var started = time.Now()

func Elapsed() time.Duration {
	return time.Since(started)
}
