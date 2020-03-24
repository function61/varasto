// Client for accessing Varasto server
package stoclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/function61/gokit/ossignal"
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
			panicIfError(wrapWithStopSupport(func(ctx context.Context) error {
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
			panicIfError(err)

			wd, err := NewWorkdirLocation(cwd)
			panicIfError(err)

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
			panicIfError(wrapWithStopSupport(func(ctx context.Context) error {
				cwd, err := os.Getwd()
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
					panicIfError(snapshotter.Release(*snapshot))
				}()

				// now read the workdir from within the snapshot (and not the actual cwd)
				wd, err := NewWorkdirLocation(snapshot.OriginInSnapshotPath)
				if err != nil {
					return err
				}

				return push(ctx, wd)
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
			panicIfError(wrapWithStopSupport(func(ctx context.Context) error {
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
			panicIfError(wrapWithStopSupport(func(ctx context.Context) error {
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
	ctx, cancel := context.WithCancel(context.Background())

	stopWaitingForSignals := make(chan interface{}, 1)
	defer close(stopWaitingForSignals)

	go func() {
		select {
		case sig := <-ossignal.InterruptOrTerminate():
			log.Printf("got %s; stopping", sig)
			cancel()
		case <-stopWaitingForSignals:
		}
	}()

	return fn(ctx)
}
