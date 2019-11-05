package stoclient

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
)

func bulkUploadScriptEntrypoint() *cobra.Command {
	rm := false

	cmd := &cobra.Command{
		Use:   "bulkuploadscript [parentDirectory]",
		Short: "Generates a shell script to adopt & push all subdirectories as collections",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			panicIfError(bulkUploadScriptGenerate(args[0], rm, os.Stdout))
		},
	}

	cmd.Flags().BoolVarP(&rm, "rm", "", rm, "Whether to remove uploaded collections")

	return cmd
}

func bulkUploadScriptGenerate(parentDirectory string, rm bool, out io.Writer) error {
	maybeRm := ""
	if rm {
		maybeRm = `sto rm "$dir"`
	}

	if _, err := fmt.Fprintf(out, `set -eu

parentDirId="%s"

one() {
	local dir="$1"

	(cd "$dir" && sto adopt -- "$parentDirId" && sto push)

	%s
}

`, parentDirectory, maybeRm); err != nil {
		return err
	}

	dentries, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}

	for _, dentry := range dentries {
		if !dentry.IsDir() {
			continue
		}

		if _, err := fmt.Fprintf(out, "one \"%s\"\n", dentry.Name()); err != nil {
			return err
		}
	}

	return nil
}
