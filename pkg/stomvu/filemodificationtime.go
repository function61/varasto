package stomvu

// Sorts files based on their modification time.

import (
	"os"

	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func fileModificationTimeEntrypoint() *cobra.Command {
	doIt := false
	yearMonth := false

	cmd := &cobra.Command{
		Use:   "file-modtime",
		Short: "Sort files based on their modification time",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(func() error {
				if yearMonth {
					return runOrExplainPlan(fileModificationTimeYearMonth, doIt)
				} else {
					return runOrExplainPlan(fileModificationTimeYear, doIt)
				}
			}())
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")
	cmd.Flags().BoolVarP(&yearMonth, "year-month", "", yearMonth, "Sort <year>/<month>/")

	return cmd
}

// sort <year>/
func fileModificationTimeYear(name string) string {
	return fileModificationTimeInternal(name, "2006")
}

// sort <year>/<month>/
func fileModificationTimeYearMonth(name string) string {
	return fileModificationTimeInternal(name, "2006/01")
}

func fileModificationTimeInternal(name string, datePattern string) string {
	stat, err := os.Stat(name)
	if err != nil {
		panic(err)
	}

	if stat.IsDir() { // directories don't make sense here
		return ""
	}

	// intentionally not UTC
	return stat.ModTime().Format(datePattern)
}
