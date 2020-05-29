package stodebug

import (
	"fmt"
	"os"

	"github.com/function61/varasto/pkg/blobstore/localfsblobstore"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	debug := &cobra.Command{
		Use:   "debug",
		Short: `Debug utilities`,
	}

	debug.AddCommand(&cobra.Command{
		Use:   "localfsblobstore-path [blobRef]",
		Short: "Format FS path from BlobRef",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref, err := stotypes.BlobRefFromHex(args[0])
			exitIfError(err)

			fmt.Println(localfsblobstore.RefToPath(*ref, "/"))
		},
	})

	return debug
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
