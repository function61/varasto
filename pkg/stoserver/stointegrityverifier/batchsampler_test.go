package stointegrityverifier

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
)

func TestNoSampler(t *testing.T) {
	sample, err := CreateSampler(nil)
	assert.Ok(t, err)

	assert.Assert(t, sample(stotypes.BlobRef{0x00, 0x00, 0x00, 0x00}) == true)
	assert.Assert(t, sample(stotypes.BlobRef{0xFF, 0xFF, 0xFF, 0xFF}) == true)
}

func TestSampler(t *testing.T) {
	sampleSpec := "11"
	sample, err := CreateSampler(&sampleSpec)
	assert.Ok(t, err)

	assert.Assert(t, sample(stotypes.BlobRef{0x00, 0x00, 0x00, 0x00}) == false)
	assert.Assert(t, sample(stotypes.BlobRef{0x00, 0x00, 0x00, 0b01}) == false)
	assert.Assert(t, sample(stotypes.BlobRef{0x00, 0x00, 0x00, 0b10}) == false)
	assert.Assert(t, sample(stotypes.BlobRef{0x00, 0x00, 0x00, 0b11}) == true)
	assert.Assert(t, sample(stotypes.BlobRef{0xFF, 0xFF, 0xFF, 0xFF}) == true)
}

func TestInvalidSpec(t *testing.T) {
	sampleSpec := "wrong"
	_, err := CreateSampler(&sampleSpec)
	assert.EqualString(t, err.Error(), "invalid sampling spec. expected binary string like 01; got 'wrong'")
}
