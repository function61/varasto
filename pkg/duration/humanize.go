package duration

import (
	"math"
	"strconv"
	"time"
)

func Humanize(dur time.Duration) string {
	milliseconds := float64(dur.Milliseconds())
	seconds := int(math.Round(milliseconds / (1.0 * 1000.0)))
	minutes := int(math.Round(milliseconds / (60.0 * 1000.0)))
	hours := int(math.Round(milliseconds / (3600.0 * 1000.0)))
	days := int(math.Round(milliseconds / (86400.0 * 1000.0)))

	plural := func(num int, singular string, plural string) string {
		if num == 1 {
			return strconv.Itoa(num) + " " + singular
		} else {
			return strconv.Itoa(num) + " " + plural
		}
	}

	switch {
	case days > 0:
		return plural(days, "day", "days")
	case hours > 0:
		return plural(hours, "hour", "hours")
	case minutes > 0:
		return plural(minutes, "minute", "minutes")
	case seconds > 0:
		return plural(seconds, "second", "seconds")
	default:
		return plural(int(milliseconds), "millisecond", "milliseconds")
	}
}
