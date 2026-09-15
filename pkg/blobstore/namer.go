package blobstore

import (
	"encoding/base32"
	"encoding/base64"

	"github.com/function61/varasto/pkg/stotypes"
)

// turns a raw blob ref into a filename
type BlobNamer interface {
	Filename(ref stotypes.BlobRef) string
}

func Base64URLNamer() BlobNamer {
	return base64URLNamer{}
}

// Windows has case insensitive filesystem (sensitivity is a recent opt-in), so use the
// lowest common denominator that's better than hex encoding.
func Base32Namer() BlobNamer {
	return base32Namer{}
}

// This should yield 32 768 (1 + 2 of 5-bit symbols when using base32) directories as maximum (see test for clarification).
func ShardNamer(delegate BlobNamer) BlobNamer {
	return shardNamer{delegate: delegate}
}

var base32LowerExtendedHexNoPadding = base32.NewEncoding("0123456789abcdefghijklmnopqrstuv").WithPadding(base32.NoPadding)

type base32Namer struct{}

func (base32Namer) Filename(ref stotypes.BlobRef) string {
	return base32LowerExtendedHexNoPadding.EncodeToString([]byte(ref))
}

type base64URLNamer struct{}

func (base64URLNamer) Filename(ref stotypes.BlobRef) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

type shardNamer struct {
	delegate BlobNamer
}

func (s shardNamer) Filename(ref stotypes.BlobRef) string {
	filename := s.delegate.Filename(ref)
	return filename[0:1] + "/" + filename[1:3] + "/" + filename[3:]
}
