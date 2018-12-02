package bupclient

import (
	"fmt"
	"github.com/function61/bup/pkg/fssnapshot"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func cloneEntrypoint() *cobra.Command {
	rev := ""

	cmd := &cobra.Command{
		Use:   "clone [collectionId] [dirName]",
		Short: "Downloads a collection from server to workdir",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			dirName := ""
			if len(args) > 1 {
				dirName = args[1]
			}

			parentDir, err := os.Getwd()
			panicIfError(err)

			panicIfError(clone(args[0], rev, parentDir, dirName))
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
					"(none)")
			}
		},
	}
}

func pushEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Uploads a collection from workdir to server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			panicIfError(err)

			// take filesystem snapshot, so our reads within the file tree are atomic
			snapshotter := fssnapshot.NullSnapshotter()
			// snapshotter := fssnapshot.PlatformSpecificSnapshotter()
			snapshot, err := snapshotter.Snapshot(cwd)
			panicIfError(err)

			defer func() { // always release snapshot
				panicIfError(snapshotter.Release(*snapshot))
			}()

			// now read the workdir from within the snapshot (and not the actual cwd)
			wd, err := NewWorkdirLocation(snapshot.OriginInSnapshotPath)
			panicIfError(err)

			panicIfError(push(wd))
		},
	}
}

func stEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "st",
		Short: "Shows working directory status compared to the parent revision",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			panicIfError(err)

			wd, err := NewWorkdirLocation(cwd)
			panicIfError(err)

			ch, err := computeChangeset(wd)
			panicIfError(err)

			for _, created := range ch.FilesCreated {
				fmt.Printf("+ %s\n", created.Path)
			}

			for _, updated := range ch.FilesUpdated {
				fmt.Printf("M %s\n", updated.Path)
			}

			for _, deleted := range ch.FilesDeleted {
				fmt.Printf("- %s\n", deleted)
			}
		},
	}
}

func mkEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "mk [collectionName]",
		Short: "Creates a new collection",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			panicIfError(err)

			panicIfError(mk(cwd, args[0]))
		},
	}
}

func Entrypoints() []*cobra.Command {
	return []*cobra.Command{
		mkEntrypoint(),
		stEntrypoint(),
		cloneEntrypoint(),
		logEntrypoint(),
		pushEntrypoint(),
		configInitEntrypoint(),
	}
}
