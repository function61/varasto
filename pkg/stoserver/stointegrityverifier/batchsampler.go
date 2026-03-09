package stointegrityverifier

// With sampling we can select only a subset of the blobs to visit.
// https://en.wikipedia.org/wiki/Sampling_(statistics)

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/function61/varasto/pkg/stotypes"
)

// answers whether we should visit a blob
type batchSampler func(stotypes.BlobRef) bool

func CreateSampler(sampleSpecificationMaybe *string) (batchSampler, error) {
	if sampleSpecification := sampleSpecificationMaybe; sampleSpecification != nil {
		// bit string like `1111` to number (`15`)
		num, err := strconv.ParseUint(*sampleSpecification, 2, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid sampling spec. expected binary string like 01; got '%s'", *sampleSpecification)
		}
		bitCount := len(*sampleSpecification)

		return prefixSampler(uint32(num), uint8(bitCount)), nil
	} else {
		return func(_ stotypes.BlobRef) bool { return true }, nil
	}
}

// only accepts blob refs that start with a specific bit pattern. that means that we can accept blob refs starting with:
// - `0b0` => accepts 1/2 of refs
// - `0b00` => accepts 1/4 of refs
// - `0b000` => accepts 1/8 of refs
// - and so on...
//
// the interesting property of this, as opposed to something like using random sampling for acceptance is that
// this is deterministic based on blob ref and thus we can integrity verify first batch today and next back next week and
// we are guaranteed that the next batch won't re-visit blobs from first batch.
//
// let's take the 1/4 acceptance as example. we have four batches: 1) `0b00` 2) `0b01` 3) `0b10` 4) `0b11`
//
// we could visit those four batches in four different weeks to guaranteed visit all blobs (except those added later to earlier batches' "partitions")
func prefixSampler(value uint32, bitCount uint8) batchSampler {
	return func(blobRef stotypes.BlobRef) bool {
		// counter-intuitively: use big-endian encoding to *not* filter on the very first bits of the blob ref, because
		// the verifier is iterating the blobs ordered on the blob ref. we don't want us to consecutively ignore a large
		// portion of the scan but instead we want to accept blobs uniformly over time
		blobRefUint32 := binary.BigEndian.Uint32(blobRef)
		return blobRefUint32&bitmask(bitCount) == value
	}
}

// `1` => `0b1`
// `3` => `0b111`
// `8` => `0b11111111`
func bitmask(n uint8) uint32 {
	return (1 << n) - 1
}
