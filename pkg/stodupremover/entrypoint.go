package stodupremover

import (
	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	dupremover := &cobra.Command{
		Use:   "dupremover",
		Short: `Remove duplicate files (files already in Varasto server)`,
	}

	dupremover.AddCommand(&cobra.Command{
		Use:   "refresh-db",
		Short: `Refresh the duplicate detector database`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(refreshDatabase(
				osutil.CancelOnInterruptOrTerminate(nil)))
		},
	})

	dupremover.AddCommand(scanEntry())

	dupremover.AddCommand(removeEmptyDirsEntrypoint())

	return dupremover
}

func scanEntry() *cobra.Command {
	acceptOutdatedDB := false
	removeDuplicates := false

	cmd := &cobra.Command{
		Use:   "scan",
		Short: `Scan current directory tree for items existing in Varasto`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(scan(removeDuplicates, acceptOutdatedDB))
		},
	}

	cmd.Flags().BoolVarP(&acceptOutdatedDB, "accept-outdated-db", "a", acceptOutdatedDB, "Accept dangerously outdated DB")
	cmd.Flags().BoolVarP(&removeDuplicates, "rm", "", removeDuplicates, "Actually remove duplicates, instead of only reporting")

	return cmd
}
