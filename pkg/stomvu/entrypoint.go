package stomvu

import (
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/spf13/cobra"
	"os"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mvu",
		Short: `Renaming utils ("mv utils") for photos, TV series etc.`,
	}

	cmd.AddCommand(epEntrypoint())
	cmd.AddCommand(photoEntrypoint())

	return cmd
}

func epEntrypoint() *cobra.Command {
	doIt := false

	cmd := &cobra.Command{
		Use:   "ep",
		Short: "Renames episodes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			plan, err := computePlan(episodeFromFilename)
			panicIfError(err)

			if doIt {
				panicIfError(executePlan(plan))
			} else {
				explainPlan(plan, os.Stdout)
			}
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")

	return cmd
}

func photoEntrypoint() *cobra.Command {
	doIt := false

	cmd := &cobra.Command{
		Use:   "photo",
		Short: "Renames photos & videos",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			plan, err := computePlan(photoDateFromFilename)
			panicIfError(err)

			if doIt {
				panicIfError(executePlan(plan))
			} else {
				explainPlan(plan, os.Stdout)
			}
		},
	}

	cmd.Flags().BoolVarP(&doIt, "do", "", doIt, "Whether to execute the plan or run a dry run")

	return cmd
}

func episodeFromFilename(input string) string {
	result := seasonepisodedetector.Detect(input)
	if result == nil {
		return ""
	}

	return result.String()
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
