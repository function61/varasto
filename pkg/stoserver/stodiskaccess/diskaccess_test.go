package stodiskaccess

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/minio/sha256-simd"
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

func (t *testDbAccess) QueryBlobExists(ref stotypes.BlobRef) (bool, error) {
	_, exists := t.metaStore[ref.AsHex()]
	return exists, nil
}

func (t *testDbAccess) QueryCollectionEncryptionKeyForNewBlobs(collId string) (string, []byte, error) {
	// for ease of testing, we'll derive each blob's encryption key by xor'ing root
	// encryption key and blob's sha256. this would not be kosher for production!
	collIdHashed := sha256.Sum256([]byte(collId))
	return collId, xorSlices(collIdHashed[:], t.rootEncryptionKey), nil
}

func (t *testDbAccess) QueryBlobCrc32(ref stotypes.BlobRef) ([]byte, error) {
	if meta, found := t.metaStore[ref.AsHex()]; found {
		return meta.ExpectedCrc32, nil
	}

	return nil, os.ErrNotExist
}

func (t *testDbAccess) QueryBlobMetadata(ref stotypes.BlobRef, kenvs []stotypes.KeyEnvelope) (*BlobMeta, error) {
	if meta, found := t.metaStore[ref.AsHex()]; found {
		return meta, nil
	}

	return nil, os.ErrNotExist
}

func (t *testDbAccess) WriteBlobReplicated(ref stotypes.BlobRef, volumeId int) error {
	return nil
}

func (t *testDbAccess) WriteBlobCreated(meta *BlobMeta, size int) error {
	// QueryCollectionEncryptionKeyForNewBlobs() returns collection id as encryption key id,
	// so here we can re-compute our testing encryption key by using EncryptionKeyId as collId
	_, encryptionKey, err := t.QueryCollectionEncryptionKeyForNewBlobs(meta.EncryptionKeyId)
	if err != nil {
		return err
	}

	// we monkey-patch EncryptionKey here so QueryBlobMetadata() doesn't return nil
	// EncryptionKey. really ugly.
	meta.EncryptionKey = encryptionKey

	t.metaStore[meta.Ref.AsHex()] = meta
	return nil
}

type testData struct {
	blobStorage  *testingBlobStorage
	testDbAccess *testDbAccess
	diskAccess   *Controller
}

func setupDefault() *testData {
	return setup(rootEncryptionKeyA)
}

func setup(encKey []byte) *testData {
	blobStorage := createVolume("2v2IQMfhcpc", 10)

	tda := &testDbAccess{
		encKey,
		map[string]*BlobMeta{}}

	diskAccess := New(tda)

	panicIfError(mount(1, blobStorage, diskAccess))

	return &testData{blobStorage, tda, diskAccess}
}

func TestWriteToUnknownVolume(t *testing.T) {
	s := setupDefault()
	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	err := s.diskAccess.WriteBlob(2, "dummyCollId", *ref, strings.NewReader("The quick brown fox jumps over the lazy dog"), true)

	assert.EqualString(t, err.Error(), "volume 2 not found")
}

func TestWriteDigestMismatch(t *testing.T) {
	s := setupDefault()
	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	err := s.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader("xxx The quick brown fox jumps over the lazy dog"), true)

	assert.EqualString(t, err.Error(), "hashVerifyReader: digest mismatch")
}

func TestWriteAndRead(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)

	// then let's try to read it

	_, err := test.diskAccess.Fetch(*ref, []stotypes.KeyEnvelope{}, 2)

	assert.EqualString(
		t,
		err.Error(),
		"volume 2 not found")

	contentReader, err := test.diskAccess.Fetch(*ref, []stotypes.KeyEnvelope{}, 1)
	assert.Assert(t, err == nil)
	defer contentReader.Close()

	content, err := ioutil.ReadAll(contentReader)
	assert.Assert(t, err == nil)

	assert.EqualString(t, string(content), "The quick brown fox jumps over the lazy dog")
}

func TestWriteSameFileWithTwoDifferentEncryptionKeys(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	encryptedWithA := setup(rootEncryptionKeyA)
	encryptedWithB := setup(rootEncryptionKeyB)

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, encryptedWithA.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)
	assert.Assert(t, encryptedWithB.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)

	assert.EqualString(t, md5Hex(encryptedWithA.blobStorage.files[sha256OfQuickBrownFox]), "9ed295ab8a5c1a4f8e759db4408dc767")
	assert.EqualString(t, md5Hex(encryptedWithB.blobStorage.files[sha256OfQuickBrownFox]), "b004754ff58ef8dfcb541c97cbea54c8")
}

func TestCannotWriteSameBlobTwice(t *testing.T) {
	contentToStore := "The quick brown fox jumps over the lazy dog"

	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)

	assert.EqualString(t, md5Hex(test.blobStorage.files[sha256OfQuickBrownFox]), "9ed295ab8a5c1a4f8e759db4408dc767")

	// cannot write same blob metadata twice
	assert.EqualString(
		t,
		test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true).Error(),
		"WriteBlob() already exists: d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
}

func TestCompressionMaybeCompressible(t *testing.T) {
	testCompressionInternal(t, true)
}

func TestCompressionNotCompressible(t *testing.T) {
	testCompressionInternal(t, false)
}

func testCompressionInternal(t *testing.T, maybeCompressible bool) {
	text := "The quick brown fox jumps over the lazy dog"
	text4x := text + text + text + text
	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(text), maybeCompressible) == nil)

	meta, err := test.testDbAccess.QueryBlobMetadata(*ref, nil)
	assert.Assert(t, err == nil)

	// this does not compress well
	assert.Assert(t, !meta.IsCompressed)
	assert.Assert(t, meta.RealSize == 43)
	assert.Assert(t, meta.SizeOnDisk == 43)

	ref2, _ := stotypes.BlobRefFromHex(sha256Hex([]byte(text4x)))

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref2, strings.NewReader(text4x), maybeCompressible) == nil)

	meta, err = test.testDbAccess.QueryBlobMetadata(*ref2, nil)
	assert.Assert(t, err == nil)

	if maybeCompressible {
		assert.Assert(t, meta.IsCompressed)
		assert.Assert(t, meta.RealSize == 4*43)
		assert.Assert(t, meta.SizeOnDisk == 70)
	} else {
		assert.Assert(t, !meta.IsCompressed)
		assert.Assert(t, meta.RealSize == 4*43)
		assert.Assert(t, meta.SizeOnDisk == 4*43)
	}

	reader, err := test.diskAccess.Fetch(*ref2, []stotypes.KeyEnvelope{}, 1)
	assert.Assert(t, err == nil)
	defer reader.Close()

	// test decompression
	content, err := ioutil.ReadAll(reader)
	assert.Assert(t, err == nil)

	assert.EqualString(t, string(content), text4x)
}

func TestRoutingCost(t *testing.T) {
	/*	We'll end up with volumes and their routing costs:

		1 => 10
		2 => 30
		3 => 20
	*/
	test := setupDefault()
	panicIfError(mount(2, createVolume("6P5rgMCeGsA", 30), test.diskAccess))
	panicIfError(mount(3, createVolume("xGili5d64vw", 20), test.diskAccess))

	bestVolumeId := func(volumeIds []int) int {
		best, err := test.diskAccess.BestVolumeId(volumeIds)
		panicIfError(err)
		return best
	}

	assert.Assert(t, bestVolumeId([]int{1, 2, 3}) == 1)
	assert.Assert(t, bestVolumeId([]int{1, 2}) == 1)
	assert.Assert(t, bestVolumeId([]int{1, 3}) == 1)
	assert.Assert(t, bestVolumeId([]int{2, 3}) == 3)
	assert.Assert(t, bestVolumeId([]int{3, 1}) == 1)
}

func TestReplication(t *testing.T) {
	test := setupDefault()
	firstStore := test.blobStorage

	secondBlobStore := createVolume("6P5rgMCeGsA", 10)
	panicIfError(mount(2, secondBlobStore, test.diskAccess))

	contentToStore := "The quick brown fox jumps over the lazy dog"

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)

	_, secondHasIt := secondBlobStore.files[sha256OfQuickBrownFox]

	assert.Assert(t, !secondHasIt)

	assert.Assert(t, test.diskAccess.Replicate(context.TODO(), 1, 2, *ref) == nil)

	_, secondHasIt = secondBlobStore.files[sha256OfQuickBrownFox]

	assert.Assert(t, secondHasIt)

	assert.Assert(
		t,
		bytes.Equal(firstStore.files[sha256OfQuickBrownFox], secondBlobStore.files[sha256OfQuickBrownFox]))
}

func TestTryReplicateRottenData(t *testing.T) {
	test := setupDefault()
	firstStore := test.blobStorage

	secondBlobStore := createVolume("6P5rgMCeGsA", 10)
	panicIfError(mount(2, secondBlobStore, test.diskAccess))

	contentToStore := "The quick brown fox jumps over the lazy dog"

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader(contentToStore), true) == nil)

	_, secondHasIt := secondBlobStore.files[sha256OfQuickBrownFox]

	assert.Assert(t, !secondHasIt)

	// make bits rot
	firstStore.files[sha256OfQuickBrownFox][3] = 0xff

	assert.EqualString(
		t,
		test.diskAccess.Replicate(context.TODO(), 1, 2, *ref).Error(),
		"hashVerifyReader: digest mismatch")
}

func TestScrubbing(t *testing.T) {
	test := setupDefault()

	ref, _ := stotypes.BlobRefFromHex(sha256OfQuickBrownFox)

	assert.Assert(t, test.diskAccess.WriteBlob(1, "dummyCollId", *ref, strings.NewReader("The quick brown fox jumps over the lazy dog"), true) == nil)

	_, err := test.diskAccess.Scrub(*ref, 1)
	assert.Assert(t, err == nil)

	// now corrupt one byte on the "disk"
	test.blobStorage.files[sha256OfQuickBrownFox][10] = 0xFF

	_, err = test.diskAccess.Scrub(*ref, 1)
	assert.EqualString(t, err.Error(), "hashVerifyReader: digest mismatch")
}

func TestTryMountIncorrectVolume(t *testing.T) {
	ctx := context.TODO()

	test := setupDefault()

	secondBlobStore := createVolume("6P5rgMCeGsA", 10)

	// volume is not yet initialized
	assert.EqualString(
		t,
		test.diskAccess.Mount(ctx, 2, secondBlobStore.uuid, secondBlobStore).Error(),
		"volume descriptor not found")

	assert.Assert(t, test.diskAccess.Initialize(ctx, secondBlobStore.uuid, secondBlobStore) == nil)

	// cannot re-initialize
	assert.EqualString(
		t,
		test.diskAccess.Initialize(ctx, secondBlobStore.uuid, secondBlobStore).Error(),
		"cannot initialize because verifyOnDiskVolumeUuid: <nil>")

	// now try mounting with wrong UUID
	assert.EqualString(
		t,
		test.diskAccess.Mount(ctx, 2, "wrongUuid", secondBlobStore).Error(),
		"unexpected volume UUID: 6P5rgMCeGsA")

	// correct UUID works
	assert.Assert(t, test.diskAccess.Mount(ctx, 2, secondBlobStore.uuid, secondBlobStore) == nil)
}

func TestVolumeDescriptorRef(t *testing.T) {
	assert.EqualString(t, volumeDescriptorRef.AsHex(), "0000000000000000000000000000000000000000000000000000000000000000")
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
	uuid        string
	files       map[string][]byte
	routingCost int
}

func mount(
	volId int,
	tbs *testingBlobStorage,
	dam *Controller,
) error {
	ctx := context.TODO()

	if err := dam.Initialize(ctx, tbs.uuid, tbs); err != nil {
		return err
	}

	if err := dam.Mount(ctx, volId, tbs.uuid, tbs); err != nil {
		return err
	}

	return nil
}

func createVolume(uuid string, routingCost int) *testingBlobStorage {
	return &testingBlobStorage{
		uuid:        uuid,
		files:       map[string][]byte{},
		routingCost: routingCost,
	}
}

func (t *testingBlobStorage) RoutingCost() int {
	return t.routingCost
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

	for k := range a {
		c[k] = a[k] ^ b[k]
	}

	return c
}

func TestXorSlices(t *testing.T) {
	a := []byte{0x01, 0x00}
	b := []byte{0x11, 0x01}

	assert.Assert(t, bytes.Equal(xorSlices(a, b), []byte{0x10, 0x01}))
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
