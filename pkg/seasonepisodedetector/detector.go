// Extracts season & episode numbers for TV show filenames
package seasonepisodedetector

import (
	"regexp"
	"strconv"
)

type Result struct {
	Season  string
	Episode string
}

func (d *Result) SeasonDesignation() string {
	return "S" + d.Season
}

func (d *Result) String() string {
	return d.SeasonDesignation() + "E" + d.Episode
}

func (d *Result) LaxEqual(other Result) bool {
	var err error

	// converts string to int while keeping track of any error occurring
	c := func(in string) int {
		parsed, errAtoi := strconv.Atoi(in)
		if errAtoi != nil {
			err = errAtoi
		}

		return parsed
	}

	return c(d.Season) == c(other.Season) && c(d.Episode) == c(other.Episode) && err == nil
}

var detectSeasonEpisodeRe = regexp.MustCompile("[Ss]([0-9]+)[Ee]([0-9]+)")

var detectSeasonEpisodeStupidRe = regexp.MustCompile("([0-9]+)[Xx]([0-9]+)")

func Detect(filename string) *Result {
	result := detectSeasonEpisodeRe.FindStringSubmatch(filename)
	if result == nil {
		result = detectSeasonEpisodeStupidRe.FindStringSubmatch(filename)
	}
	if result == nil {
		return nil
	}

	return &Result{
		Season:  result[1],
		Episode: result[2],
	}
}
