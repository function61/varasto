// +build !windows

package varastofuse

import (
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/spf13/cobra"
	"log"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE integration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "serve <collectionId>",
		Short: "Mounts a Varasto collection via FUSE as a local directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()

			go func() {
				log.Printf("Received %s; stopping", <-ossignal.InterruptOrTerminate())
				workers.StopAllWorkersAndWait()
			}()

			if err := fuseServe(args[0], "/samba/joonas/varasto", workers.Stopper()); err != nil {
				panic(err)
			}

			log.Printf("Stopped successfully")
		},
	})

	return cmd
}
