package stoutils

import (
	"io"
	"path"
	"strings"

	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/minio/sha256-simd"
)

// this should not be called from anywhere other than DiskAccessManager and varastoclient
func BlobHashVerifier(reader io.Reader, br stotypes.BlobRef) io.Reader {
	return hashverifyreader.New(reader, sha256.New(), br.AsSha256Sum())
}

// there's gonna be lots of these
var NewCollectionID = longID
var NewDirectoryID = longID

// there's going to be comparatively few of these
// (changeset IDs are unique within a collection)
var NewCollectionChangesetID = shortID
var NewVolumeMountID = shortID
var NewVolumeUUID = longID
var NewNodeID = shortID
var NewClientID = shortID
var NewIntegrityVerificationJobID = shortID
var NewReplicationPolicyID = shortID
var NewEncryptionKeyID = longID
var NewKeyEncryptionKeyID = shortID
var NewAPIKeySecret = cryptoLongID

func shortID() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(3)
}

func longID() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(8)
}

func cryptoLongID() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(32)
}

func IsMaybeCompressible(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".jpg", ".jpeg", ".gif", ".png", ".mp4", ".mkv", ".avi", ".mp3":
		return false
	default:
		return true
	}
}
