package stotypes

import (
	"errors"
)

var (
	ErrBlobNotAccessibleOnThisNode = errors.New("blob not accessible on this node")
)
