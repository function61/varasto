package seasonepisodedetector

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestDetect(t *testing.T) {
	cs := []struct {
		input  string
		expect string
	}{
		{
			input:  "[ www.Speed.Cd ] - Grand.Designs.S12E06.720p.HDTV.x264-C4TV",
			expect: "S12E06",
		},
		{
			input:  "[ www.Speed.Cd ] - Grand.Designs.s12e6.720p.HDTV.x264-C4TV",
			expect: "S12E6",
		},
		{
			input:  "[ www.Speed.Cd ] - Grand.Designs.s12e.720p.HDTV.x264-C4TV",
			expect: "nomatch",
		},
		{
			input:  "Simpsons 07x01 - Who Shot Mr Burns (Part 2) [rl]",
			expect: "S07E01",
		},
	}

	for _, c := range cs {
		t.Run(c.input, func(t *testing.T) {
			res := Detect(c.input)

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
