package stoclient

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func rm(ctx context.Context, path string) error {
	dir, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// will error out if not a workdir
	wd, err := NewWorkdirLocation(dir)
	if err != nil {
		return err
	}

	ch, err := computeChangeset(ctx, wd, NewBlobDiscoveredNoopListener())
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
			panicIfError(wrapWithStopSupport(func(ctx context.Context) error {
				return rm(ctx, args[0])
			}))
		},
	}
}
