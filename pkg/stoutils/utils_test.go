package stoutils

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestIsMaybeCompressible(t *testing.T) {
	tcs := []struct {
		filename                  string
		expectedMaybeCompressible bool
	}{
		{
			"hello.txt",
			true,
		},
		{
			"main.go",
			true,
		},
		{
			"Unknown.fileformat",
			true,
		},
		{
			"movie.mp4",
			false,
		},
		{
			"DCIM20190930_331.jpg",
			false,
		},
		{
			"DCIM20190930_331.JPG",
			false,
		},
		{
			"DCIM20190930_331.jpeg",
			false,
		},
		{
			"killitwithfire.gif",
			false,
		},
		{
			"Dexter-S03E02.mkv",
			false,
		},
		{
			"2001: A Space Odyssey.AVI",
			false,
		},
		{
			"Rick Astley - Never Gonna Give You Up.mp3",
			false,
		},
	}

	for _, tc := range tcs {
		tc := tc // pin
		t.Run(tc.filename, func(t *testing.T) {
			assert.Assert(t, IsMaybeCompressible(tc.filename) == tc.expectedMaybeCompressible)
		})
	}
}
