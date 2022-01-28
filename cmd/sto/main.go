package main

import (
	"os"

	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stodebug"
	"github.com/function61/varasto/pkg/stofuse/stofuseentrypoint"
	"github.com/function61/varasto/pkg/stomvu"
	"github.com/function61/varasto/pkg/stoserver"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   `Varasto CLI: sto ("STOrage without the rage")`,
		Version: dynversion.Version,
		// hide the default "completion" subcommand from polluting UX (it can still be used). https://github.com/spf13/cobra/issues/1507
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	}

	// client's commands are at the root level somewhat unhygienically for convenience's
	// sake (since the client CLI commands are used most often).
	for _, entrypoint := range stoclient.Entrypoints() {
		rootCmd.AddCommand(entrypoint)
	}

	rootCmd.AddCommand(stoserver.Entrypoint())
	rootCmd.AddCommand(stofuseentrypoint.Entrypoint())
	rootCmd.AddCommand(stomvu.Entrypoint())
	rootCmd.AddCommand(stodebug.Entrypoint())

	osutil.ExitIfError(rootCmd.Execute())
}
