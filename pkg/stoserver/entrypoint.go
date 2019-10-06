// The server component main package for Varasto
package stoserver

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/gokit/systemdinstaller"
	"github.com/function61/varasto/pkg/logtee"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
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
			logTail := logtee.NewStringTail(50)

			// writes to upstream all end up in the sink, but logTail.Snapshot() only
			// returns the last "capacity" lines
			rootLogger := logex.StandardLoggerTo(logtee.NewLineSplitterTee(os.Stderr, func(line string) {
				logTail.Write(line)
			}))

			workers := stopper.NewManager()
			go func(logger *log.Logger) {
				logex.Levels(logger).Info.Printf(
					"Got %s; stopping",
					<-ossignal.InterruptOrTerminate())

				workers.StopAllWorkersAndWait()
			}(logex.Prefix("main", rootLogger))

			panicIfError(runServer(rootLogger, logTail, workers.Stopper()))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "dbimport",
		Short: "Imports database",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			scf, err := readServerConfigFile()
			if err != nil {
				panic(err)
			}

			if err := stodbimportexport.Import(os.Stdin, scf.DbLocation); err != nil {
				panic(err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Installs systemd unit file to make Varasto start on system boot",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			serviceFile := systemdinstaller.SystemdServiceFile(
				"varasto",
				"Varasto server",
				systemdinstaller.Args("server"),
				systemdinstaller.Docs("https://github.com/function61/varasto", "https://function61.com/"),
				systemdinstaller.RequireNetworkOnline)

			if err := systemdinstaller.Install(serviceFile); err != nil {
				panic(err)
			} else {
				fmt.Println(systemdinstaller.GetHints(serviceFile))
			}
		},
	})

	return cmd
}
