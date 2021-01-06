package stomvu

import (
	"os"
	"path/filepath"

	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/spf13/cobra"
)

func tvEntrypoint() *cobra.Command {
	doIt := false

	cmd := &cobra.Command{
		Use:   "tv",
		Short: "Renames TV episodes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			plan, err := ComputePlan("./", episodeFromFilename)
			osutil.ExitIfError(err)

			if doIt {
				osutil.ExitIfError(ExecutePlan(plan))
			} else {
				explainPlan(plan, os.Stdout)
			}
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")

	return cmd
}

// logic is already tested in seasonepisodedetector package
func episodeFromFilename(input string) string {
	result := seasonepisodedetector.Detect(input)
	if result == nil {
		return ""
	}

	// "S03/S03E07"
	return filepath.Join(result.SeasonDesignation(), result.String())
}
