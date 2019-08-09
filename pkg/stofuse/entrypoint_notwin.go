// +build !windows

package stofuse

import (
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/spf13/cobra"
	"log"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE integration",
	}

	unmountFirst := false

	serveCmd := &cobra.Command{
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

			if err := func() error {
				conf, err := stoclient.ReadConfig()
				if err != nil {
					return err
				}

				return fuseServe(sigs, *conf, unmountFirst, workers.Stopper())
			}(); err != nil {
				panic(err)
			}

			log.Printf("Stopped successfully")
		},
	}

	serveCmd.Flags().BoolVarP(&unmountFirst, "unmount-first", "u", unmountFirst, "Umount the mount-path first (maybe unclean shutdown previously)")

	cmd.AddCommand(serveCmd)

	return cmd
}
