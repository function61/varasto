// `gokit` backports. things that exist in newer version of gokit, but which we cannot update to yet.
package gokitbp

import (
	"time"
)

var (
	DefaultReadHeaderTimeout = 60 * time.Second
)
