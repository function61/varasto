// +build !windows

// FUSE adapter for interfacing with Varasto from filesystem
package stofuse

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE integration",
	}

	addr := ":8689"
	unmountFirst := false

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Mounts a FUSE-based FS to serve collections from Varasto",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()

			rootLogger := logex.StandardLogger()
			logl := logex.Levels(rootLogger)

			conf, err := stoclient.ReadConfig()
			if err != nil {
				panic(err)
			}

			go func() {
				// wait for stdin EOF (or otherwise broken pipe)
				_, _ = io.Copy(ioutil.Discard, os.Stdin)

				logl.Error.Println("parent process died (detected by closed stdin) - stopping")

				workers.StopAllWorkersAndWait() // safe to call two times, concurrently
			}()

			go func() {
				logl.Info.Printf("got %s; stopping", <-ossignal.InterruptOrTerminate())

				workers.StopAllWorkersAndWait()
			}()

			sigs := newSigs()

			go func(stop *stopper.Stopper) {
				logl.Info.Printf("starting to listen on %s", addr)

				if err := rpcServe(addr, sigs, stop); err != nil {
					panic(err)
				}
			}(workers.Stopper())

			if err := fuseServe(sigs, *conf, unmountFirst, workers.Stopper(), logl); err != nil {
				panic(err)
			}

			logl.Info.Println("stopped")
		},
	}

	serveCmd.Flags().StringVarP(&addr, "addr", "", addr, "Address to listen on")
	serveCmd.Flags().BoolVarP(&unmountFirst, "unmount-first", "u", unmountFirst, "Umount the mount-path first (maybe unclean shutdown previously)")

	cmd.AddCommand(serveCmd)

	return cmd
}
