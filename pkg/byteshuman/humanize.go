// Formats byte amounts into human readable format
package byteshuman

import (
	"fmt"
)

const (
	B   = 1
	kiB = 1024 * B
	MiB = 1024 * kiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

func Humanize(num uint64) string {
	switch {
	case num >= PiB:
		return fmt.Sprintf("%.02f PiB", float64(num)/PiB)
	case num >= TiB:
		return fmt.Sprintf("%.02f TiB", float64(num)/TiB)
	case num >= GiB:
		return fmt.Sprintf("%.02f GiB", float64(num)/GiB)
	case num >= MiB:
		return fmt.Sprintf("%.02f MiB", float64(num)/MiB)
	case num >= kiB:
		return fmt.Sprintf("%.02f kiB", float64(num)/kiB)
	default:
		return fmt.Sprintf("%d B", num)
	}
}
