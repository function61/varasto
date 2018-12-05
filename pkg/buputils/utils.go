package buputils

import (
	"crypto/sha256"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/hashverifyreader"
	"io"
)

func BlobHashVerifier(reader io.Reader, br buptypes.BlobRef) io.Reader {
	return hashverifyreader.New(reader, sha256.New(), br.AsSha256Sum())
}

// there's gonna be lots of these
var NewCollectionId = longId

// there's going to be comparatively few of these
// (changeset IDs are unique within a collection)
var NewCollectionChangesetId = shortId
var NewVolumeMountId = shortId
var NewNodeId = shortId
var NewClientId = shortId

func shortId() string {
	return cryptorandombytes.Base64Url(3)
}

func longId() string {
	return cryptorandombytes.Base64Url(8)
}
