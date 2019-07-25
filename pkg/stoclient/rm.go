package stoclient

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func rmEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "rm",
		Short: "Removes a working directory, but only if remote has full state",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := os.Getwd()
			panicIfError(err)

			wd, err := NewWorkdirLocation(dir)
			panicIfError(err)

			ch, err := computeChangeset(wd)
			panicIfError(err)

			if ch.AnyChanges() {
				fmt.Println("Refusing to delete workdir because it has changes")
				os.Exit(1)
			}

			// switch away from the dir we are removing
			userDir, err := os.UserHomeDir()
			panicIfError(err)
			panicIfError(os.Chdir(userDir))

			panicIfError(os.RemoveAll(dir))
		},
	}
}
