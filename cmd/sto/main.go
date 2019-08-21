package main

import (
	"fmt"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stofuse"
	"github.com/function61/varasto/pkg/stomvu"
	"github.com/function61/varasto/pkg/stoserver"
	"github.com/function61/varasto/pkg/stothumb"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   `Varasto CLI: sto ("storage without the rage")`,
		Version: dynversion.Version,
	}

	for _, entrypoint := range stoclient.Entrypoints() {
		rootCmd.AddCommand(entrypoint)
	}

	rootCmd.AddCommand(stoserver.Entrypoint())
	rootCmd.AddCommand(stofuse.Entrypoint())
	rootCmd.AddCommand(stothumb.Entrypoint())
	rootCmd.AddCommand(stomvu.Entrypoint())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
