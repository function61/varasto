// +build !windows

package stofuse

import (
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stofuse/stofuseclient"
	"github.com/spf13/cobra"
	"log"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE integration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "mount <collectionId>",
		Short: "Mounts a collection by id",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := stofuseclient.New().Mount(args[0]); err != nil {
				panic(err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "unmount <collectionId>",
		Short: "Unmounts a collection by id",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := stofuseclient.New().Unmount(args[0]); err != nil {
				panic(err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Mounts a FUSE-based FS to serve collections from Varasto",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()

			go func() {
				log.Printf("Received %s; stopping", <-ossignal.InterruptOrTerminate())
				workers.StopAllWorkersAndWait()
			}()

			sigs := newSigs()

			go rpcServe(sigs, workers.Stopper())

			if err := fuseServe(sigs, "/samba/joonas/varasto", workers.Stopper()); err != nil {
				panic(err)
			}

			log.Printf("Stopped successfully")
		},
	})

	return cmd
}
