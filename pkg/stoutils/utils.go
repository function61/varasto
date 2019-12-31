package stoutils

import (
	"crypto/sha256"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/varasto/pkg/stotypes"
	"io"
	"path"
	"strings"
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
	return randomBase64UrlWithoutLeadingDash(3)
}

func longId() string {
	return randomBase64UrlWithoutLeadingDash(8)
}

func cryptoLongId() string {
	return randomBase64UrlWithoutLeadingDash(32)
}

// CLI arguments beginning with dash are problematic (which base64 URL variant can produce),
// so we'll be nice guys and guarantee that the ID won't start with one.
func randomBase64UrlWithoutLeadingDash(length int) string {
	id := cryptorandombytes.Base64Url(length)

	if id[0] == '-' {
		// try again. the odds should exponentially decrease for recursion level to increase
		return randomBase64UrlWithoutLeadingDash(length)
	}

	return id
}

func IsMaybeCompressible(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".jpg", ".jpeg", ".gif", ".png", ".mp4", ".mkv", ".avi", ".mp3":
		return false
	default:
		return true
	}
}
