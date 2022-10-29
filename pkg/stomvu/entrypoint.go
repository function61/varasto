package stomvu

import (
	"os"

	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mvu",
		Short: `Renaming utils ("mv utils") for photos, TV series etc.`,
	}

	cmd.AddCommand(tvEntrypoint())
	cmd.AddCommand(photoEntrypoint())
	cmd.AddCommand(customMonthlyPatternEntrypoint())

	return cmd
}

func runOrExplainPlan(targetFn func(string) string, doIt bool) error {
	plan, err := ComputePlan("./", targetFn)
	if err != nil {
		return err
	}

	if doIt {
		return ExecutePlan(plan)
	} else {
		explainPlan(plan, os.Stdout)
		return nil
	}
}
