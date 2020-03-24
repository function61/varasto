package main

import (
	"fmt"
	"os"

	"github.com/function61/gokit/dynversion"
	"github.com/function61/varasto/pkg/stoclient"
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
	}

	// client's commands are at the root level somewhat unhygienically for convenience's
	// sake (since the client CLI commands are used most often).
	for _, entrypoint := range stoclient.Entrypoints() {
		rootCmd.AddCommand(entrypoint)
	}

	rootCmd.AddCommand(stoserver.Entrypoint())
	rootCmd.AddCommand(stofuseentrypoint.Entrypoint())
	rootCmd.AddCommand(stomvu.Entrypoint())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
