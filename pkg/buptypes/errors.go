package buptypes

import (
	"errors"
)

var (
	ErrChunkMetadataNotFound       = errors.New("chunk metadata not found")
	ErrChunkAlreadyExists          = errors.New("chunk already exists")
	ErrBlobNotAccessibleOnThisNode = errors.New("blob not accessible on this node")
	ErrUnknownCommand              = errors.New("unknown command")
	// ErrUnexpectedResponseStatusCode = errors.New("unexpected response code")
	ErrBadBlobRef = errors.New("bad blob ref")
)
