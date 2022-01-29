// Client for accessing Varasto server
package stoclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/fssnapshot"
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

				client, err := ReadConfig()
				if err != nil {
					return err
				}

				return client.Clone(ctx, args[0], rev, parentDir, dirName)
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

			client, err := ReadConfig()
			osutil.ExitIfError(err)

			wd, err := client.NewWorkdirLocation(cwd)
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
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}

				client, err := ReadConfig()
				if err != nil {
					return err
				}

				// take filesystem snapshot, so our reads within the file tree are atomic
				snapshotter := fssnapshot.NullSnapshotter()
				// snapshotter := fssnapshot.PlatformSpecificSnapshotter()
				snapshot, err := snapshotter.Snapshot(cwd)
				if err != nil {
					return err
				}

				defer func() { // always release snapshot
					osutil.ExitIfError(snapshotter.Release(*snapshot))
				}()

				// now read the workdir from within the snapshot (and not the actual cwd)
				wd, err := client.NewWorkdirLocation(snapshot.OriginInSnapshotPath)
				if err != nil {
					return err
				}

				return Push(
					ctx,
					wd,
					textUiUploadProgressOutputIfInTerminal())
			}))
		},
	}

	cmd.AddCommand(bulkUploadScriptEntrypoint())

	return cmd
}

func pushOneEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "pushone [collectionId] [file]",
		Short: "Uploads a single file to a collection",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				return pushOne(ctx, args[0], args[1])
			}))
		},
	}
}

func stEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "st",
		Short: "Shows working directory status compared to the parent revision",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(func(ctx context.Context) error {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}

				client, err := ReadConfig()
				if err != nil {
					return err
				}

				wd, err := client.NewWorkdirLocation(cwd)
				if err != nil {
					return err
				}

				ch, err := ComputeChangeset(ctx, wd, NewBlobDiscoveredNoopListener())
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
		stEntrypoint(),
		rmEntrypoint(),
		cloneEntrypoint(),
		logEntrypoint(),
		pushEntrypoint(),
		pushOneEntrypoint(),
		configInitEntrypoint(),
		configPrintEntrypoint(),
	}
}

func wrapWithStopSupport(fn func(ctx context.Context) error) error {
	return fn(osutil.CancelOnInterruptOrTerminate(logex.StandardLogger()))
}
