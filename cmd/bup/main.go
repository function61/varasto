package main

import (
	"fmt"
	"github.com/function61/bup/pkg/bupclient"
	"github.com/function61/bup/pkg/bupserver"
	"github.com/spf13/cobra"
	"os"
)

var version = "dev" // replaced dynamically at build time

func main() {
	rootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Your friendly distributed NAS",
		Version: version,
	}

	for _, entrypoint := range bupclient.Entrypoints() {
		rootCmd.AddCommand(entrypoint)
	}

	rootCmd.AddCommand(bupserver.Entrypoint())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
