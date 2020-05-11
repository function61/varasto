package stotypes

import (
	"errors"
)

var (
	ErrBlobNotAccessibleOnThisNode = errors.New("blob not accessible on this node")
	ErrBadBlobRef                  = errors.New("bad blob ref")
)
