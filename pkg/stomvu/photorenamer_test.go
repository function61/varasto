package stomvu

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestDetectPhotoVideoDate(t *testing.T) {
	cs := []struct {
		input  string
		expect string
	}{
		{"IMG_20180526_151345.jpg", "2018-05 - Unsorted"},
		{"VID_20190626_151345.jpg", "2019-06 - Unsorted"},
		{"20170429_194919.mp4", "2017-04 - Unsorted"},
		{"20170627_203226.jpg", "2017-06 - Unsorted"},
		{"IMG_20180526666_151345.jpg", "nomatch"},
	}

	for _, c := range cs {
		t.Run(c.input, func(t *testing.T) {
			res := detectPhotoVideoDate(c.input)

			var output string
			if res != nil {
				output = res.String()
			} else {
				output = "nomatch"
			}

			assert.EqualString(t, output, c.expect)
		})
	}
}
