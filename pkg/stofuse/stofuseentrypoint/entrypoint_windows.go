package stofuseentrypoint

import (
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE does not work in Windows",
	}
}
