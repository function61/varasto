package stothumb

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
)

func Entrypoint() *cobra.Command {
	addr := ":8688"

	cmd := &cobra.Command{
		Use:   "thumbserver",
		Short: "Starts the thumbnail server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()

			rootLogger := logex.StandardLogger()
			logl := logex.Levels(rootLogger)

			go func() {
				// wait for stdin EOF (or otherwise broken pipe)
				_, _ = io.Copy(ioutil.Discard, os.Stdin)

				logl.Error.Println("parent process died (detected by closed stdin) - stopping")

				workers.StopAllWorkersAndWait() // safe to call two times, concurrently
			}()

			go func() {
				logl.Info.Printf("got %s; stopping", <-ossignal.InterruptOrTerminate())

				workers.StopAllWorkersAndWait() // safe to call two times, concurrently
			}()

			if err := runServer(addr, rootLogger, workers.Stopper()); err != nil {
				panic(err)
			}

			logl.Info.Println("stopped")
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "", addr, "Address to listen on")

	return cmd
}
