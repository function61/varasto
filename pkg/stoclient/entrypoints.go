// Client for accessing Varasto server
package stoclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func cloneEntrypoint() *cobra.Command {
	rev := ""

	cmd := &cobra.Command{
		Use:   "clone [collectionId] [dirName]",
		Short: "Downloads a collection from server to workdir",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				dirName := ""
				if len(args) > 1 {
					dirName = args[1]
				}

				parentDir, err := os.Getwd()
				if err != nil {
					return err
				}

				return clone(ctx, args[0], rev, parentDir, dirName)
			}))
		},
	}

	cmd.Flags().StringVarP(&rev, "rev", "r", rev, "Revision to clone")

	return cmd
}

func logEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show changeset log",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			osutil.ExitIfError(err)

			wd, err := NewWorkdirLocation(cwd)
			osutil.ExitIfError(err)

			for _, item := range wd.manifest.Collection.Changesets {
				fmt.Printf(
					"changeset:   %s\ndate:        %s\nsummary:     %s\n\n",
					item.ID,
					item.Created.Format(time.RFC822Z),
					fmt.Sprintf(
						"(no description) %d create(s) %d update(s) %d delete(s)",
						len(item.FilesCreated),
						len(item.FilesUpdated),
						len(item.FilesDeleted)))
			}
		},
	}
}

func pushEntrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Uploads a collection from workdir to server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(pushCurrentWorkdir))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "one [collectionId] [file]",
		Short: "Uploads a single file to a collection",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				return pushOne(ctx, args[0], args[1])
			}))
		},
	})

	cmd.AddCommand(bulkUploadScriptEntrypoint())

	return cmd
}

func statusEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Shows working directory status compared to the parent revision",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}

				wd, err := NewWorkdirLocation(cwd)
				if err != nil {
					return err
				}

				ch, err := computeChangeset(ctx, wd, NewBlobDiscoveredNoopListener())
				if err != nil {
					return err
				}

				for _, created := range ch.FilesCreated {
					fmt.Printf("+ %s\n", created.Path)
				}

				for _, updated := range ch.FilesUpdated {
					fmt.Printf("M %s\n", updated.Path)
				}

				for _, deleted := range ch.FilesDeleted {
					fmt.Printf("- %s\n", deleted)
				}

				return nil
			}))
		},
	}
}

func Entrypoints() []*cobra.Command {
	return []*cobra.Command{
		adoptEntrypoint(),
		statusEntrypoint(),
		rmEntrypoint(),
		cloneEntrypoint(),
		logEntrypoint(),
		pushEntrypoint(),
		configEntrypoint(),
	}
}

func wrapWithStopSupport(fn func(ctx context.Context) error) error {
	return fn(osutil.CancelOnInterruptOrTerminate(logex.StandardLogger()))
}
