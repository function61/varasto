package stodupremover

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func removeEmptyDirs(path string, dry bool) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				if dry {
					fmt.Printf("would remove empty dir: %s\n", path)
				} else {
					fmt.Printf("removing empty dir: %s\n", path)

					// TODO: not sure if Walk() will be ok with removing currently walking item?
					if err := os.Remove(path); err != nil {
						//nolint:staticcheck // capitalization in error message ok here
						return fmt.Errorf("Rmdir: %v", err)
					}
				}
			}
		}

		return nil
	})
}

func removeEmptyDirsEntrypoint() *cobra.Command {
	really := false

	cmd := &cobra.Command{
		Use:   "removeemptydirs [path]",
		Short: "Remove empty leaf directories (run this multiple times to remove one level at a time)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(removeEmptyDirs(args[0], !really))
		},
	}

	cmd.Flags().BoolVarP(&really, "really", "", really, "Really remove the files. Without this a dry run is performed.")

	return cmd
}
