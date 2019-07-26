package stoserver

import (
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the server component",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := log.New(os.Stderr, "", log.LstdFlags)

			workers := stopper.NewManager()
			go func(logger *log.Logger) {
				logex.Levels(logger).Info.Printf(
					"Got %s; stopping",
					<-ossignal.InterruptOrTerminate())

				workers.StopAllWorkersAndWait()
			}(logex.Prefix("main", rootLogger))

			panicIfError(runServer(rootLogger, workers.Stopper()))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "dbimport",
		Short: "Imports database",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := importDb(os.Stdin); err != nil {
				panic(err)
			}
		},
	})

	return cmd
}
