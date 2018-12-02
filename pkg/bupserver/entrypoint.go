package bupserver

import (
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/spf13/cobra"
	"log"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Starts the server component",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()
			go func() {
				log.Printf("Got %s; stopping", <-ossignal.InterruptOrTerminate())
				workers.StopAllWorkersAndWait()
			}()

			panicIfError(runServer(workers.Stopper()))
		},
	}
}
