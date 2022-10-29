package stomvu

import (
	"path/filepath"
	"regexp"
	"time"

	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func customMonthlyPattern(reString string, dateformat string) func(name string) string {
	re := regexp.MustCompile(reString)

	return func(name string) string {
		result := re.FindStringSubmatch(name)
		if result == nil {
			return ""
		}

		ts, err := time.Parse(dateformat, result[1])
		if err != nil {
			return ""
		}

		return filepath.Join(
			ts.Format("2006"),
			ts.Format("01"))
	}
}

func customMonthlyPatternEntrypoint() *cobra.Command {
	doIt := false

	cmd := &cobra.Command{
		Use:   "custom-monthly [regexp] [dateformat]",
		Short: "Custom date pattern for moving to monthly folders. The first capture group must be the timestamp",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(runOrExplainPlan(customMonthlyPattern(args[0], args[1]), doIt))
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")

	return cmd
}
