package stomvu

import (
	"regexp"
)

type PhotoResult struct {
	DateString string
}

func (p *PhotoResult) String() string {
	// <year>-<month>
	return p.DateString[0:4] + "-" + p.DateString[4:6] + " - Unsorted"
}

// IMG_20180526_151345.jpg
// VID_20180526_151345.mp4
var detectPhotoVideoDateRe = regexp.MustCompile("^((?:IMG|VID)_)?([0-9]{8})_")

func detectPhotoVideoDate(filename string) *PhotoResult {
	result := detectPhotoVideoDateRe.FindStringSubmatch(filename)
	if result == nil {
		return nil
	}

	return &PhotoResult{
		DateString: result[2],
	}
}

func photoDateFromFilename(name string) string {
	result := detectPhotoVideoDate(name)
	if result == nil {
		return ""
	}

	return result.String()
}
