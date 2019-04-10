package main

import (
	"fmt"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/varasto/pkg/varastoclient"
	"github.com/function61/varasto/pkg/varastofuse"
	"github.com/function61/varasto/pkg/varastoserver"
	"github.com/function61/varasto/pkg/varastothumb"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Your friendly distributed NAS",
		Version: dynversion.Version,
	}

	for _, entrypoint := range varastoclient.Entrypoints() {
		rootCmd.AddCommand(entrypoint)
	}

	rootCmd.AddCommand(varastoserver.Entrypoint())
	rootCmd.AddCommand(varastofuse.Entrypoint())
	rootCmd.AddCommand(varastothumb.Entrypoint())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
