// +build !windows

// This entrypoint is in own package, so we don't need to sprinkle conditional compilation
// all around the base "stofuse" package because it doesn't compile on Windows
package stofuseentrypoint

import (
	"github.com/function61/varasto/pkg/stofuse"
)

var Entrypoint = stofuse.Entrypoint
