package stomvu

import (
	"regexp"

	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
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

func PhotoOrVideoDateFromFilename(name string) string {
	result := detectPhotoVideoDate(name)
	if result == nil {
		return ""
	}

	return result.String()
}

func photoEntrypoint() *cobra.Command {
	doIt := false

	cmd := &cobra.Command{
		Use:   "photo",
		Short: "Organize photos & videos to '<year>-<month> - Unsorted' subdirectories",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(runOrExplainPlan(PhotoOrVideoDateFromFilename, doIt))
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")

	return cmd
}
