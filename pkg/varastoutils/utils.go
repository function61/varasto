package varastoutils

import (
	"crypto/sha256"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/varasto/pkg/varastotypes"
	"io"
)

func BlobHashVerifier(reader io.Reader, br varastotypes.BlobRef) io.Reader {
	return hashverifyreader.New(reader, sha256.New(), br.AsSha256Sum())
}

// there's gonna be lots of these
var NewCollectionId = longId
var NewDirectoryId = longId

// there's going to be comparatively few of these
// (changeset IDs are unique within a collection)
var NewCollectionChangesetId = shortId
var NewVolumeMountId = shortId
var NewVolumeUuid = longId
var NewNodeId = shortId
var NewClientId = shortId
var NewIntegrityVerificationJobId = shortId

func shortId() string {
	return cryptorandombytes.Base64Url(3)
}

func longId() string {
	return cryptorandombytes.Base64Url(8)
}
