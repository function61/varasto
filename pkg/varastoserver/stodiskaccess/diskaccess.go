package stodiskaccess

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"github.com/function61/gokit/hashverifyreader"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/blobdriver"
	"github.com/function61/varasto/pkg/varastotypes"
	"github.com/function61/varasto/pkg/varastoutils"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
)

// diskaccess ties together DB metadata read/write in addition to writing to disk
type Controller struct {
	metadataStore   MetadataStore
	mountedDrivers  map[int]blobdriver.Driver // only mounted drivers
	legacyDriverIds []int
}

func New(metadataStore MetadataStore) *Controller {
	return &Controller{
		metadataStore,
		map[int]blobdriver.Driver{},
		[]int{},
	}
}

// call only during server boot (these are not threadsafe)
func (d *Controller) Define(volumeId int, driver blobdriver.Driver, legacy bool) {
	if _, exists := d.mountedDrivers[volumeId]; exists {
		panic("driver for volumeId already defined")
	}

	d.mountedDrivers[volumeId] = driver

	if legacy {
		d.legacyDriverIds = append(d.legacyDriverIds, volumeId)
	}
}

func (d *Controller) IsMounted(volumeId int) bool {
	_, mounted := d.mountedDrivers[volumeId]
	return mounted
}

// in theory we wouldn't need to do this since we could do a Fetch()-followed by Store(),
// but we can optimize by just transferring the raw on-disk format
func (d *Controller) Replicate(fromVolumeId int, toVolumeId int, ref varastotypes.BlobRef) error {
	// TODO: use lower-level APIs so we don't have to decrypt-and-encrypt
	stream, err := d.Fetch(ref, fromVolumeId)
	if err != nil {
		return err
	}
	defer stream.Close()

	meta, err := d.metadataStore.QueryBlobMetadata(ref)
	if err != nil { // expecting this
		return fmt.Errorf("Replicate() QueryBlobMetadata: %v", err)
	}

	meta, err = d.storeInternal(toVolumeId, meta.Optional.EncryptionKeyOfColl, ref, stream)
	if err != nil {
		return err
	}

	return d.metadataStore.WriteBlobReplicated(meta, toVolumeId)
}

func (d *Controller) WriteBlob(volumeId int, collId string, ref varastotypes.BlobRef, content io.Reader) error {
	// since we're writing a blob (and not replicating), for safety we'll check that we haven't
	// seen this blob before
	if _, err := d.metadataStore.QueryBlobMetadata(ref); err != os.ErrNotExist { // expecting this
		if err != nil {
			return err // some other error
		}

		return fmt.Errorf("WriteBlob() already exists: %s", ref.AsHex())
	}

	// this is going to take relatively long time, so we can't keep
	// a transaction open
	meta, err := d.storeInternal(volumeId, collId, ref, content)
	if err != nil {
		return err
	}

	if err := d.metadataStore.WriteBlobCreated(meta, volumeId); err != nil {
		return fmt.Errorf("WriteBlob() DB write: %v", err)
	}

	return nil
}

// does everything about storing except for the database write
func (d *Controller) storeInternal(volumeId int, collId string, ref varastotypes.BlobRef, content io.Reader) (*BlobMeta, error) {
	driver, isLegacy, err := d.driverFor(volumeId)
	if err != nil {
		return nil, err
	}

	readCounter := writeCounter{}
	verifiedContent := readCounter.Tee(varastoutils.BlobHashVerifier(content, ref))

	encryptionKey, err := d.metadataStore.QueryCollectionEncryptionKey(collId)
	if err != nil {
		return nil, err
	}

	// need copy of verifiedContent here for use of legacy driver
	verifiedContentCopyForLegacy := &bytes.Buffer{}

	blobEncrypted, err := encryptAndCompressBlob(io.TeeReader(verifiedContent, verifiedContentCopyForLegacy), encryptionKey, ref)
	if err != nil {
		return nil, err
	}

	mkBlobMeta := func() *BlobMeta {
		return &BlobMeta{
			Ref:      ref,
			RealSize: int32(readCounter.BytesWritten()),
			Optional: &BlobMetaOptional{
				SizeOnDisk:          int32(len(blobEncrypted.CiphertextMaybeCompressed)),
				IsCompressed:        blobEncrypted.Compressed,
				EncryptionKeyOfColl: collId,
				EncryptionKey:       encryptionKey,
				ExpectedCrc32:       blobEncrypted.Crc32,
			},
		}
	}

	if isLegacy {
		if err := driver.RawStore(ref, verifiedContentCopyForLegacy); err != nil {
			return nil, err
		}

		return mkBlobMeta(), nil
	}

	if err := driver.RawStore(ref, bytes.NewReader(blobEncrypted.CiphertextMaybeCompressed)); err != nil {
		return nil, fmt.Errorf("storing blob into volume %d failed: %v", volumeId, err)
	}

	return mkBlobMeta(), nil
}

// does decrypt(optional_decompress(blobOnDisk))
// verifies blob integrity for you!
func (d *Controller) Fetch(ref varastotypes.BlobRef, volumeId int) (io.ReadCloser, error) {
	driver, isLegacy, err := d.driverFor(volumeId)
	if err != nil {
		return nil, err
	}

	if isLegacy {
		body, err := driver.RawFetch(ref)
		if err != nil {
			return nil, err
		}

		return ioutil.NopCloser(varastoutils.BlobHashVerifier(body, ref)), nil
	}

	meta, err := d.metadataStore.QueryBlobMetadata(ref)
	if err != nil {
		return nil, err
	}

	body, err := driver.RawFetch(meta.Ref)
	if err != nil {
		return nil, err
	}
	// body.Close() will be called by readCloseWrapper

	// reads crc32-verified ciphertext which contains maybe_gzipped(plaintext)
	crc32VerifiedCiphertextReader := hashverifyreader.New(body, crc32.NewIEEE(), meta.Optional.ExpectedCrc32)

	aesDecrypter, err := aes.NewCipher(meta.Optional.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("Fetch() AES cipher: %v", err)
	}

	decrypted := &cipher.StreamReader{S: cipher.NewCTR(aesDecrypter, deriveIvFromBlobRef(meta.Ref)), R: crc32VerifiedCiphertextReader}

	// assume no compression ..
	uncompressedReader := io.Reader(decrypted)

	if meta.Optional.IsCompressed { // .. but decompress here if assumption incorrect
		gzipReader, err := gzip.NewReader(decrypted)
		if err != nil {
			return nil, fmt.Errorf("Fetch() gzip: %v", err)
		}

		uncompressedReader = gzipReader
	}

	blobDecryptedUncompressed := varastoutils.BlobHashVerifier(uncompressedReader, meta.Ref)

	return &readCloseWrapper{blobDecryptedUncompressed, body}, nil
}

// currently looks for the first ID mounted on this node, but in the future could use richer heuristics:
// - is the HDD currently spinning
// - best latency & bandwidth
func (d *Controller) BestVolumeId(volumeIds []int) (int, error) {
	for _, volumeId := range volumeIds {
		if d.IsMounted(volumeId) {
			return volumeId, nil
		}
	}

	return 0, varastotypes.ErrBlobNotAccessibleOnThisNode
}

// runs a scrubbing job for a blob in a given volume to detect errors
// https://en.wikipedia.org/wiki/Data_scrubbing
// we could actually just do a Fetch() but that would require access to the encryption keys.
// this way we can verify on-disk integrity without encryption keys.
func (d *Controller) Scrub(ref varastotypes.BlobRef, volumeId int) (int64, error) {
	driver, isLegacy, err := d.driverFor(volumeId)
	if err != nil {
		return 0, err
	}

	if isLegacy {
		stream, err := driver.RawFetch(ref)
		if err != nil {
			return 0, err
		}
		defer stream.Close()

		bytesRead, err := io.Copy(ioutil.Discard, varastoutils.BlobHashVerifier(stream, ref))
		return bytesRead, err
	}

	meta, err := d.metadataStore.QueryBlobMetadata(ref)
	if err != nil {
		return 0, err
	}

	body, err := driver.RawFetch(meta.Ref)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	verifiedReader := hashverifyreader.New(body, crc32.NewIEEE(), meta.Optional.ExpectedCrc32)

	bytesRead, err := io.Copy(ioutil.Discard, verifiedReader)
	return bytesRead, err
}

func (d *Controller) driverFor(volumeId int) (blobdriver.Driver, bool, error) {
	driver, found := d.mountedDrivers[volumeId]
	if !found {
		return nil, false, fmt.Errorf("volume %d not found", volumeId)
	}

	return driver, sliceutil.ContainsInt(d.legacyDriverIds, volumeId), nil
}

func encrypt(key []byte, plaintext io.Reader, br varastotypes.BlobRef) ([]byte, error) {
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

// does encrypt(maybe_compress(plaintext))
func encryptAndCompressBlob(contentReader io.Reader, encryptionKey []byte, ref varastotypes.BlobRef) (*blobResult, error) {
	content, err := ioutil.ReadAll(contentReader)
	if err != nil {
		return nil, err
	}

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

	contentMaybeCompressed := content

	if wellCompressible {
		contentMaybeCompressed = compressed.Bytes()
	}

	ciphertextMaybeCompressed, err := encrypt(encryptionKey, bytes.NewReader(contentMaybeCompressed), ref)
	if err != nil {
		return nil, err
	}

	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(ciphertextMaybeCompressed))

	return &blobResult{
		CiphertextMaybeCompressed: ciphertextMaybeCompressed,
		Compressed:                wellCompressible,
		Crc32:                     crc,
	}, nil
}

func deriveIvFromBlobRef(br varastotypes.BlobRef) []byte {
	return br.AsSha256Sum()[0:aes.BlockSize]
}
