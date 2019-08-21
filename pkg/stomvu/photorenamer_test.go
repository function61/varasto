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
		{
			input:  "IMG_20180526_151345.jpg",
			expect: "2018-05 - Unsorted",
		},
		{
			input:  "VID_20190626_151345.jpg",
			expect: "2019-06 - Unsorted",
		},
		{
			input:  "20170429_194919.mp4",
			expect: "2017-04 - Unsorted",
		},
		{
			input:  "20170627_203226.jpg",
			expect: "2017-06 - Unsorted",
		},
		{
			input:  "IMG_20180526666_151345.jpg",
			expect: "nomatch",
		},
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
