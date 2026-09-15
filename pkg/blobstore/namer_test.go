package blobstore

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
)

func TestNamers(t *testing.T) {
	ref, err := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
	assert.Ok(t, err)

	assert.EqualString(t,
		Base32Namer().Filename(*ref),
		"qukfnco7qu098qeajaub021e9u6lckf4dkudmthd0b8budu9sm90")
	assert.EqualString(t,
		Base64URLNamer().Filename(*ref),
		"16j7swfXgJRpypq8sAguT41WUeRtPNt2LQLQvzfJ5ZI")
	assert.EqualString(t,
		ShardNamer(Base32Namer()).Filename(*ref),
		"q/uk/fnco7qu098qeajaub021e9u6lckf4dkudmthd0b8budu9sm90")
}
