// diskaccess ties together DB metadata read/write in addition to writing to disk
package stodiskaccess

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/blobstore"
	"github.com/function61/varasto/pkg/mutexmap"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
)

var (
	ErrVolumeDescriptorNotFound = errors.New("volume descriptor not found")
	volumeDescriptorRef         = stotypes.BlobRef{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
)

type Controller struct {
	metadataStore  MetadataStore
	mountedDrivers map[int]blobstore.Driver // only mounted drivers
	routingCosts   map[int]int              // volume id => cost. lower (local disks) is better than higher (remote disks)
	writingBlobs   *mutexmap.M
}

type volumeDescriptor struct {
	VolumeUUID string `json:"volume_uuid"`
}

func New(metadataStore MetadataStore) *Controller {
	return &Controller{
		metadataStore,
		map[int]blobstore.Driver{},
		map[int]int{},
		mutexmap.New(),
	}
}

// call only during server boot (this is not threadsafe)
func (d *Controller) Mount(ctx context.Context, volumeID int, expectedVolumeUUID string, driver blobstore.Driver) error {
	if err := d.Mountable(ctx, volumeID, expectedVolumeUUID, driver); err != nil {
		return err
	}

	d.mountedDrivers[volumeID] = driver
	d.routingCosts[volumeID] = driver.RoutingCost()

	return nil
}

// mount command currently wants to check if volume would be mountable without actually mounting it
func (d *Controller) Mountable(ctx context.Context, volumeID int, expectedVolumeUUID string, driver blobstore.Driver) error {
	if _, exists := d.mountedDrivers[volumeID]; exists {
		return errors.New("driver for volumeId already defined")
	}

	if err := d.verifyOnDiskVolumeUUID(ctx, driver, expectedVolumeUUID); err != nil {
		return err
	}

	return nil
}

func (d *Controller) IsMounted(volumeID int) bool {
	_, mounted := d.mountedDrivers[volumeID]
	return mounted
}

// in theory we wouldn't need to do this since we could do a Fetch()-followed by Store(),
// but we can optimize by just transferring the raw on-disk format
func (d *Controller) Replicate(ctx context.Context, fromVolumeID int, toVolumeID int, ref stotypes.BlobRef) error {
	fromDriver, err := d.driverFor(fromVolumeID)
	if err != nil {
		return err
	}

	toDriver, err := d.driverFor(toVolumeID)
	if err != nil {
		return err
	}

	expectedCrc32, err := d.metadataStore.QueryBlobCrc32(ref)
	if err != nil {
		return fmt.Errorf("Replicate() QueryBlobCrc32: %v", err)
	}

	rawContent, err := fromDriver.RawFetch(ctx, ref)
	if err != nil {
		return err
	}
	defer rawContent.Close()

	crc32VerifiedReader := hashverifyreader.New(rawContent, crc32.NewIEEE(), expectedCrc32)

	if err := toDriver.RawStore(ctx, ref, crc32VerifiedReader); err != nil {
		return err
	}

	return d.metadataStore.WriteBlobReplicated(ref, toVolumeID)
}

func (d *Controller) WriteBlob(
	volumeID int,
	collID string,
	ref stotypes.BlobRef,
	content io.Reader,
	maybeCompressible bool,
) error {
	return d.WriteBlobNoVerify(
		volumeID,
		collID,
		ref,
		stoutils.BlobHashVerifier(content, ref),
		maybeCompressible)
}

func (d *Controller) WriteBlobNoVerify(
	volumeID int,
	collID string,
	ref stotypes.BlobRef,
	content io.Reader,
	maybeCompressible bool,
) error {
	// FIXME: this will lock for a really long time if the HTTP connection breaks (TODO: benchmark for how long).
	// should we have some kind of timeoutreader?
	unlock, ok := d.writingBlobs.TryLock(ref.AsHex())
	if !ok {
		return fmt.Errorf("another thread is currently writing blob[%s]", ref.AsHex())
	}
	defer unlock()

	// since we're writing a blob (and not replicating), for safety we'll check that we haven't
	// seen this blob before
	if exists, err := d.metadataStore.QueryBlobExists(ref); err != nil || exists {
		if err != nil { // error checking existence
			return err
		} else {
			// blob exists, which is unexpected
			return fmt.Errorf("WriteBlob() already exists: %s", ref.AsHex())
		}
	}

	// this is going to take relatively long time, so we can't keep
	// a transaction open

	driver, err := d.driverFor(volumeID)
	if err != nil {
		return err
	}

	readCounter := writeCounter{}
	contentCounted := readCounter.Tee(content)

	encryptionKeyID, encryptionKey, err := d.metadataStore.QueryCollectionEncryptionKeyForNewBlobs(collID)
	if err != nil {
		return err
	}

	blobEncrypted, err := encryptAndCompressBlob(contentCounted, encryptionKey, ref, maybeCompressible)
	if err != nil {
		return err
	}

	if err := driver.RawStore(context.TODO(), ref, bytes.NewReader(blobEncrypted.CiphertextMaybeCompressed)); err != nil {
		return fmt.Errorf("storing blob into volume %d failed: %v", volumeID, err)
	}

	meta := &BlobMeta{
		Ref:             ref,
		RealSize:        int32(readCounter.BytesWritten()),
		SizeOnDisk:      int32(len(blobEncrypted.CiphertextMaybeCompressed)),
		IsCompressed:    blobEncrypted.Compressed,
		EncryptionKeyID: encryptionKeyID,
		ExpectedCrc32:   blobEncrypted.Crc32,
	}

	if err := d.metadataStore.WriteBlobCreated(meta, volumeID); err != nil {
		return fmt.Errorf("WriteBlob() DB write: %v", err)
	}

	return nil
}

// does decrypt(optional_decompress(blobOnDisk))
// verifies blob integrity for you!
func (d *Controller) Fetch(ref stotypes.BlobRef, encryptionKeys []stotypes.KeyEnvelope, volumeID int) (io.ReadCloser, error) {
	driver, err := d.driverFor(volumeID)
	if err != nil {
		return nil, err
	}

	meta, err := d.metadataStore.QueryBlobMetadata(ref, encryptionKeys)
	if err != nil {
		return nil, err
	}

	body, err := driver.RawFetch(context.TODO(), ref)
	if err != nil {
		return nil, err
	}
	// body.Close() will be called by readCloseWrapper

	// reads crc32-verified ciphertext which contains maybe_gzipped(plaintext)
	crc32VerifiedCiphertextReader := hashverifyreader.New(body, crc32.NewIEEE(), meta.ExpectedCrc32)

	aesDecrypter, err := aes.NewCipher(meta.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("Fetch() AES cipher: %v", err)
	}

	decrypted := &cipher.StreamReader{S: cipher.NewCTR(aesDecrypter, deriveIvFromBlobRef(ref)), R: crc32VerifiedCiphertextReader}

	// assume no compression ..
	uncompressedReader := io.Reader(decrypted)

	if meta.IsCompressed { // .. but patch in decompression step if assumption incorrect
		gzipReader, err := gzip.NewReader(decrypted)
		if err != nil {
			return nil, fmt.Errorf("Fetch() gzip: %v", err)
		}

		uncompressedReader = gzipReader
	}

	blobDecryptedUncompressed := stoutils.BlobHashVerifier(uncompressedReader, ref)

	return &readCloseWrapper{blobDecryptedUncompressed, body}, nil
}

// currently looks for the first ID mounted on this node, but in the future could use richer heuristics:
// - is the HDD currently spinning
// - best latency & bandwidth
func (d *Controller) BestVolumeID(volumeIDs []int) (int, error) {
	lowestCost := 99
	lowestCostVolumeID := 0

	for _, volumeID := range volumeIDs {
		if !d.IsMounted(volumeID) {
			continue
		}

		cost := d.routingCosts[volumeID]

		if cost < lowestCost {
			lowestCostVolumeID = volumeID
			lowestCost = cost
		}
	}

	if lowestCostVolumeID == 0 {
		return 0, stotypes.ErrBlobNotAccessibleOnThisNode
	}

	return lowestCostVolumeID, nil
}

// runs a scrub for a blob in a given volume to detect errors
// https://en.wikipedia.org/wiki/Data_scrubbing
// we could actually just do a Fetch() but that would require access to the encryption keys.
// this way we can verify on-disk integrity without encryption keys.
func (d *Controller) Scrub(ref stotypes.BlobRef, volumeID int) (int64, error) {
	driver, err := d.driverFor(volumeID)
	if err != nil {
		return 0, err
	}

	expectedCrc32, err := d.metadataStore.QueryBlobCrc32(ref)
	if err != nil {
		return 0, err
	}

	body, err := driver.RawFetch(context.TODO(), ref)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	verifiedReader := hashverifyreader.New(body, crc32.NewIEEE(), expectedCrc32)

	bytesRead, err := io.Copy(io.Discard, verifiedReader)
	return bytesRead, err
}

// initializes volume for Varasto use by writing a volume descriptor (see verifyOnDiskVolumeUuid() for rationale)
func (d *Controller) Initialize(ctx context.Context, volumeUUID string, driver blobstore.Driver) error {
	// this error is expected to happen before initialization
	// (we check this so we know it's safe to initialize)
	if err := d.verifyOnDiskVolumeUUID(ctx, driver, volumeUUID); err != ErrVolumeDescriptorNotFound {
		return fmt.Errorf("cannot initialize because verifyOnDiskVolumeUuid: %v", err)
	}

	volDescriptor := &bytes.Buffer{}
	if err := jsonfile.Marshal(volDescriptor, &volumeDescriptor{volumeUUID}); err != nil {
		return err
	}

	return driver.RawStore(ctx, volumeDescriptorRef, volDescriptor)
}

// it's really dangerous to accidentally mount the wrong volume, so we'll keep a volume descriptor
// file at sha256=0000.. that contains the volume UUID that we can validate at mount time.
// will return ErrVolumeDescriptorNotFound if volume descriptor does not yet exist
func (d *Controller) verifyOnDiskVolumeUUID(ctx context.Context, driver blobstore.Driver, expectedVolumeUUID string) error {
	body, err := driver.RawFetch(ctx, volumeDescriptorRef)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrVolumeDescriptorNotFound
		}

		return err
	}
	defer body.Close()

	descriptor := volumeDescriptor{}
	if err := jsonfile.Unmarshal(body, &descriptor, true); err != nil {
		return err
	}

	if descriptor.VolumeUUID != expectedVolumeUUID {
		return fmt.Errorf("unexpected volume UUID: %s", descriptor.VolumeUUID)
	}

	return nil
}

func (d *Controller) driverFor(volumeID int) (blobstore.Driver, error) {
	driver, found := d.mountedDrivers[volumeID]
	if !found {
		return nil, fmt.Errorf("volume %d not found", volumeID)
	}

	return driver, nil
}

func encrypt(key []byte, plaintext io.Reader, br stotypes.BlobRef) ([]byte, error) {
	aesEncrypter, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	streamCipher := cipher.NewCTR(aesEncrypter, deriveIvFromBlobRef(br))

	var cipherText bytes.Buffer

	ciphertextWriter := &cipher.StreamWriter{S: streamCipher, W: &cipherText}

	// Copy the input to the output buffer, encrypting as we go.
	if _, err := io.Copy(ciphertextWriter, plaintext); err != nil {
		return nil, err
	}

	return cipherText.Bytes(), nil
}

// used for encryptAndCompressBlob()
type blobResult struct {
	CiphertextMaybeCompressed []byte
	Compressed                bool
	Crc32                     []byte
}

// does encrypt(maybe_compress(plaintext))
func encryptAndCompressBlob(
	contentReader io.Reader,
	encryptionKey []byte,
	ref stotypes.BlobRef,
	maybeCompressible bool,
) (*blobResult, error) {
	content, err := io.ReadAll(contentReader)
	if err != nil {
		return nil, err
	}

	contentMaybeCompressed := content
	contentIsCompressed := false

	/*	Here are perf measurements from my year 2012 CPU

		concurrency=2 | by trying compression = 25.1 MB/s
		concurrency=2 | no compression = 50.0 MB/s
		concurrency=4 | no compression = 65.7 MB/s
										66.3 MB/s
	*/
	if maybeCompressible {
		var compressed bytes.Buffer
		compressedWriter := gzip.NewWriter(&compressed)

		if _, err := compressedWriter.Write(content); err != nil {
			return nil, err
		}

		if err := compressedWriter.Close(); err != nil {
			return nil, err
		}

		compressionRatio := float64(compressed.Len()) / float64(len(content))

		wellCompressible := compressionRatio < 0.9

		if wellCompressible {
			contentMaybeCompressed = compressed.Bytes()
			contentIsCompressed = true
		}
	}

	ciphertextMaybeCompressed, err := encrypt(encryptionKey, bytes.NewReader(contentMaybeCompressed), ref)
	if err != nil {
		return nil, err
	}

	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(ciphertextMaybeCompressed))

	return &blobResult{
		CiphertextMaybeCompressed: ciphertextMaybeCompressed,
		Compressed:                contentIsCompressed,
		Crc32:                     crc,
	}, nil
}

func deriveIvFromBlobRef(br stotypes.BlobRef) []byte {
	return br.AsSha256Sum()[0:aes.BlockSize]
}
