package stoclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func rm(ctx context.Context, path string) error {
	dir, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	client, err := ReadConfig()
	if err != nil {
		return err
	}

	// will error out if not a workdir
	wd, err := client.NewWorkdirLocation(dir)
	if err != nil {
		return err
	}

	ch, err := ComputeChangeset(ctx, wd, NewBlobDiscoveredNoopListener())
	if err != nil {
		return err
	}

	if ch.AnyChanges() {
		fmt.Printf("Refusing to delete workdir '%s' because it has changes\n", path)
		os.Exit(1)
	}

	return os.RemoveAll(dir)
}

func rmEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <path>",
		Short: "Removes a local clone of collection, but only if remote has full state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				return rm(ctx, args[0])
			}))
		},
	}
}
