package stothumb

import (
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "thumbserver",
		Short: "Starts the thumbnail server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workers := stopper.NewManager()

			go func() {
				<-ossignal.InterruptOrTerminate()

				workers.StopAllWorkersAndWait()
			}()

			if err := runServer(workers.Stopper()); err != nil {
				panic(err)
			}
		},
	}
}
