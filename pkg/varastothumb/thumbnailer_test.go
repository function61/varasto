package varastothumb

import (
	"fmt"
	"github.com/function61/gokit/assert"
	"testing"
)

func TestGenThumbPath(t *testing.T) {
	assert.EqualString(t, genThumbPath([]byte{1, 2, 3, 4, 5, 6, 7, 8}), "thumbs/AQ/IDBAUGBwg.jpg")
}

func TestResizedDimensions300x533(t *testing.T) {
	tcs := []struct {
		width    int
		height   int
		expected string
	}{
		{
			16,
			16,
			"300x300",
		},
		{
			3264,
			1836,
			"300x168",
		},
		{
			1836,
			3264,
			"299x533",
		},
		{
			400,
			200,
			"300x150", // 2:1 ratio
		},
		{
			250,
			1000,
			"133x533", // 1:4 ratio
		},
	}

	for _, tc := range tcs {
		t.Run(tc.expected, func(t *testing.T) {
			w, h := resizedDimensions(tc.width, tc.height, 300, 533)
			result := fmt.Sprintf("%dx%d", w, h)

			assert.EqualString(t, result, tc.expected)
		})
	}
}
