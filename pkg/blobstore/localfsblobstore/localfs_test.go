package localfsblobstore

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
)

func TestPath(t *testing.T) {
	driver := New("APvMjudT4IQ", "/tmp/", nil)

	blobRef, err := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
	assert.Ok(t, err)

	// base32Decode("qukfnco7qu098qeajaub021e9u6lckf4dkudmthd0b8budu9sm90")
	// = hex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
	assert.EqualString(t,
		driver.getPath(*blobRef),
		"/tmp/q/uk/fnco7qu098qeajaub021e9u6lckf4dkudmthd0b8budu9sm90")
}

func TestRawStoreExistingBlob(t *testing.T) {
	driver := New("APvMjudT4IQ", t.TempDir(), nil)
	blobRef, err := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
	assert.Ok(t, err)

	expected := []byte("The quick brown fox jumps over the lazy dog")
	assert.Ok(t, driver.RawStore(context.Background(), *blobRef, bytes.NewReader(expected)))
	assert.Ok(t, driver.RawStore(context.Background(), *blobRef, bytes.NewReader(expected)))

	err = driver.RawStore(context.Background(), *blobRef, bytes.NewReader([]byte("different content")))
	assert.EqualString(t, err.Error(), "existing blob content mismatch: "+blobRef.AsHex())

	stored, err := driver.RawFetch(context.Background(), *blobRef)
	assert.Ok(t, err)
	defer stored.Close()

	actual, err := io.ReadAll(stored)
	assert.Ok(t, err)
	assert.Assert(t, bytes.Equal(actual, expected))
}
