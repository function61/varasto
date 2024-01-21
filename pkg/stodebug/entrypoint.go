package stodebug

import (
	"fmt"

	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/blobstore/localfsblobstore"
	"github.com/function61/varasto/pkg/easteregg"
	"github.com/function61/varasto/pkg/stodupremover"
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
			osutil.ExitIfError(err)

			fmt.Println(localfsblobstore.RefToPath(*ref, "/"))
		},
	})

	debug.AddCommand(easteregg.Entrypoint())
	debug.AddCommand(stodupremover.Entrypoint())

	return debug
}
