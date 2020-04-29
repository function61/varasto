package stoserver

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/themoviedbapi"
)

func TestEncodeTmdbRef(t *testing.T) {
	assert.EqualString(t, encodeTmdbRef(themoviedbapi.MediaTypeMovie, "tt7207398"), "tmdb:movie:tt7207398")
	assert.EqualString(t, encodeTmdbRef(themoviedbapi.MediaTypeTv, "tt0904208"), "tmdb:tv:tt0904208")
}

func TestDecodeInvalidTmdbRef(t *testing.T) {
	typ, _, err := decodeTmdbRef("foo")
	assert.Assert(t, err == nil)
	assert.EqualString(t, typ, "")

	_, _, err = decodeTmdbRef("tmdb:invalidtype:123")
	assert.EqualString(t, err.Error(), "unsupported tmdb type: 123")
}

func TestDecodeTmdbMovieRef(t *testing.T) {
	typ, id, err := decodeTmdbRef(encodeTmdbRef(themoviedbapi.MediaTypeMovie, "tt7207398"))
	assert.Assert(t, err == nil)
	assert.EqualString(t, typ, themoviedbapi.MediaTypeMovie)
	assert.EqualString(t, id, "tt7207398")
}

func TestDecodeTmdbTvRef(t *testing.T) {
	typ, id, err := decodeTmdbRef(encodeTmdbRef(themoviedbapi.MediaTypeTv, "tt0904208"))
	assert.Assert(t, err == nil)
	assert.EqualString(t, typ, themoviedbapi.MediaTypeTv)
	assert.EqualString(t, id, "tt0904208")
}
