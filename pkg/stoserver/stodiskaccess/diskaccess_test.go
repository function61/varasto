package stodiskaccess

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	sha256OfQuickBrownFox = "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	rootEncryptionKeyA    = []byte{0x81, 0x56, 0x65, 0x57, 0x2b, 0xdf, 0x5e, 0xdd, 0x1e, 0xe9, 0xcd, 0xca, 0xba, 0xe3, 0x98, 0x2d, 0xf9, 0x07, 0xa2, 0x72, 0xb1, 0x7d, 0xc6, 0xa6, 0x08, 0x96, 0x07, 0x8f, 0xdd, 0x33, 0x40, 0xbe}
	rootEncryptionKeyB    = []byte{0x82, 0x56, 0x65, 0x57, 0x2b, 0xdf, 0x5e, 0xdd, 0x1e, 0xe9, 0xcd, 0xca, 0xba, 0xe3, 0x98, 0x2d, 0xf9, 0x07, 0xa2, 0x72, 0xb1, 0x7d, 0xc6, 0xa6, 0x08, 0x96, 0x07, 0x8f, 0xdd, 0x33, 0x40, 0xbe}
)

type testDbAccess struct {
	rootEncryptionKey []byte
	metaStore         map[string]*BlobMeta
}

func (t *testDbAccess) QueryCollectionEncryptionKey(collId string) ([]byte, error) {
	// for ease of testing, we'll derive each blob's encryption key by xor'ing root
	// encryption key and blob's sha256. this would not be kosher for production!
	collIdHashed := sha256.Sum256([]byte(collId))
	return xorSlices(collIdHashed[:], t.rootEncryptionKey), nil
}

func (t *testDbAccess) QueryBlobMetadata(ref stotypes.BlobRef) (*BlobMeta, error) {
	if meta, found := t.metaStore[ref.AsHex()]; found {
		return meta, nil
	}

	return nil, os.ErrNotExist
}

func (t *testDbAccess) WriteBlobReplicated(meta *BlobMeta, volumeId int) error {
	return nil
}

func (t *testDbAccess) WriteBlobCreated(meta *BlobMeta, size int) error {
	t.metaStore[meta.Ref.AsHex()] = meta
	return nil
}

type testSaga struct {
	blobStorage  *testingBlobStorage
	testDbAccess *testDbAccess
	diskAccess   *Controller
}

func setupDefault() *testSaga {
	return setup(rootEncryptionKeyA, false)
}

func setup(encKey []byte, legacy bool) *testSaga {
	blobStorage := newTestingBlobStorage()

	tda := &testDbAccess{
		encKey,
		map[string]*BlobMeta{}}

	diskAccess := New(tda)

	diskAccess.Define(1, blobStorage, legacy)

	return &testSaga{blobStorage, tda, diskAccess}
}

func TestWriteToUnknownVolume(t *testing.T) {
	s := setupDefault()
	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	err := s.diskAccess.WriteBlob(2, "dummyCollId", *ref, strings.NewReader("The quick brown fox jumps over the lazy dog"))

	assert.EqualString(t, err.Error(), "volume 2 not found")
}

func TestWriteDigestMismatch(t *testing.T) {
	s := setupDefault()
	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	err := s.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader("xxx The quick brown fox jumps over the lazy dog"))

	assert.EqualString(t, err.Error(), "hashVerifyReader: digest mismatch")
}

func TestWriteAndRead(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	// then let's try to read it

	_, err := test.diskAccess.Fetch(*ref, 2)

	assert.EqualString(
		t,
		err.Error(),
		"volume 2 not found")

	contentReader, err := test.diskAccess.Fetch(*ref, 1)
	assert.Assert(t, err == nil)
	defer contentReader.Close()

	content, err := ioutil.ReadAll(contentReader)
	// assert.EqualString(t, err.Error(), "!?")
	assert.Assert(t, err == nil)

	assert.EqualString(t, string(content), "The quick brown fox jumps over the lazy dog")
}

func TestWriteSameFileWithTwoDifferentEncryptionKeys(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	encryptedWithA := setup(rootEncryptionKeyA, false)
	encryptedWithB := setup(rootEncryptionKeyB, false)

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, encryptedWithA.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)
	assert.Assert(t, encryptedWithB.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	assert.EqualString(t, md5Hex(encryptedWithA.blobStorage.files[sha256OfQuickBrownFox]), "9ed295ab8a5c1a4f8e759db4408dc767")
	assert.EqualString(t, md5Hex(encryptedWithB.blobStorage.files[sha256OfQuickBrownFox]), "b004754ff58ef8dfcb541c97cbea54c8")
}

func TestCannotWriteSameBlobTwice(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	assert.EqualString(t, md5Hex(test.blobStorage.files[sha256OfQuickBrownFox]), "9ed295ab8a5c1a4f8e759db4408dc767")

	// cannot write same blob metadata twice
	assert.EqualString(
		t,
		test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)).Error(),
		"WriteBlob() already exists: d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
}

func TestCompression(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog"
	text4x := text + text + text + text
	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(text)) == nil)

	meta, err := test.testDbAccess.QueryBlobMetadata(*ref)
	assert.Assert(t, err == nil)

	// this does not compress well
	assert.Assert(t, !meta.IsCompressed)
	assert.Assert(t, meta.RealSize == 43)
	assert.Assert(t, meta.SizeOnDisk == 43)

	ref2, _ := stotypes.BlobRefFromHex(sha256Hex([]byte(text4x)))

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref2, strings.NewReader(text4x)) == nil)

	meta, err = test.testDbAccess.QueryBlobMetadata(*ref2)
	assert.Assert(t, err == nil)

	assert.Assert(t, meta.IsCompressed)
	assert.Assert(t, meta.RealSize == 4*43)
	assert.Assert(t, meta.SizeOnDisk == 70)

	reader, err := test.diskAccess.Fetch(*ref2, 1)
	assert.Assert(t, err == nil)
	defer reader.Close()

	// test decompression
	content, err := ioutil.ReadAll(reader)
	assert.Assert(t, err == nil)

	assert.EqualString(t, string(content), text4x)
}

func TestReplicationLegacyToModern(t *testing.T) {
	test := setup(rootEncryptionKeyA, true)
	legacyStore := test.blobStorage

	modernStore := newTestingBlobStorage()
	test.diskAccess.Define(2, modernStore, false)

	contentToStore := "The quick brown fox jumps over the lazy dog"

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	assert.Assert(t, test.diskAccess.Replicate(1, 2, *ref) == nil)

	// content on legacy volume should be in plaintext
	assert.EqualString(t, string(legacyStore.files[sha256OfQuickBrownFox]), contentToStore)

	// lengths should be equal
	assert.Assert(t, len(modernStore.files[sha256OfQuickBrownFox]) == len(legacyStore.files[sha256OfQuickBrownFox]))

	// since modern volume has it encrypted, it should not be same as plaintext
	assert.Assert(t, !bytes.Equal(modernStore.files[sha256OfQuickBrownFox], legacyStore.files[sha256OfQuickBrownFox]))
}

func TestReplicationModernToLegacy(t *testing.T) {
	test := setupDefault()
	modernStore := test.blobStorage

	legacyStore := newTestingBlobStorage()
	test.diskAccess.Define(2, legacyStore, true)

	contentToStore := "The quick brown fox jumps over the lazy dog"

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	assert.Assert(t, test.diskAccess.Replicate(1, 2, *ref) == nil)

	// content on legacy volume should be in plaintext
	assert.EqualString(t, string(legacyStore.files[sha256OfQuickBrownFox]), contentToStore)

	// lengths should be equal
	assert.Assert(t, len(modernStore.files[sha256OfQuickBrownFox]) == len(legacyStore.files[sha256OfQuickBrownFox]))

	// since modern volume has it encrypted, it should not be same as plaintext
	assert.Assert(t, !bytes.Equal(modernStore.files[sha256OfQuickBrownFox], legacyStore.files[sha256OfQuickBrownFox]))
}

func TestReplicationModernToModern(t *testing.T) {
	test := setupDefault()
	firstStore := test.blobStorage

	secondBlobStore := newTestingBlobStorage()
	test.diskAccess.Define(2, secondBlobStore, false)

	contentToStore := "The quick brown fox jumps over the lazy dog"

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore)) == nil)

	_, secondHasIt := secondBlobStore.files[sha256OfQuickBrownFox]

	assert.Assert(t, !secondHasIt)

	assert.Assert(t, test.diskAccess.Replicate(1, 2, *ref) == nil)

	_, secondHasIt = secondBlobStore.files[sha256OfQuickBrownFox]

	assert.Assert(t, secondHasIt)

	assert.Assert(
		t,
		bytes.Equal(firstStore.files[sha256OfQuickBrownFox], secondBlobStore.files[sha256OfQuickBrownFox]))
}

func TestScrubbing(t *testing.T) {
	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader("The quick brown fox jumps over the lazy dog")) == nil)

	_, err := test.diskAccess.Scrub(*ref, 1)
	assert.Assert(t, err == nil)

	// now corrupt one byte on the "disk"
	test.blobStorage.files[sha256OfQuickBrownFox][10] = 0xFF

	_, err = test.diskAccess.Scrub(*ref, 1)
	assert.EqualString(t, err.Error(), "hashVerifyReader: digest mismatch")
}

func md5Hex(input []byte) string {
	sum := md5.Sum(input)
	return hex.EncodeToString(sum[:])
}

func sha256Hex(input []byte) string {
	sum := sha256.Sum256(input)
	return hex.EncodeToString(sum[:])
}

type testingBlobStorage struct {
	files map[string][]byte
}

func newTestingBlobStorage() *testingBlobStorage {
	return &testingBlobStorage{
		files: map[string][]byte{},
	}
}

func (t *testingBlobStorage) Mountable(_ context.Context) error {
	return nil
}

func (t *testingBlobStorage) RawFetch(_ context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	buf, exists := t.files[ref.AsHex()]
	if !exists {
		return nil, os.ErrNotExist
	}

	return ioutil.NopCloser(bytes.NewReader(buf)), nil
}

func (t *testingBlobStorage) RawStore(_ context.Context, ref stotypes.BlobRef, content io.Reader) error {
	buf, err := ioutil.ReadAll(content)
	if err != nil {
		return err
	}

	t.files[ref.AsHex()] = buf

	return nil
}

func xorSlices(a []byte, b []byte) []byte {
	if len(a) != len(b) {
		panic("nope")
	}

	c := make([]byte, len(a))

	for k, _ := range a {
		c[k] = a[k] ^ b[k]
	}

	return c
}

func TestXorSlices(t *testing.T) {
	a := []byte{0x01, 0x00}
	b := []byte{0x11, 0x01}

	assert.Assert(t, bytes.Equal(xorSlices(a, b), []byte{0x10, 0x01}))
}
