package stoutils

import (
	"crypto/sha256"
	"io"
	"path"
	"strings"

	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/varasto/pkg/stotypes"
)

// this should not be called from anywhere other than DiskAccessManager and varastoclient
func BlobHashVerifier(reader io.Reader, br stotypes.BlobRef) io.Reader {
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
var NewEncryptionKeyId = longId
var NewKeyEncryptionKeyId = shortId
var NewApiKeySecret = cryptoLongId

func shortId() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(3)
}

func longId() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(8)
}

func cryptoLongId() string {
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
