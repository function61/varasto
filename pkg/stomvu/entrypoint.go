package stomvu

import (
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
