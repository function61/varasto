// The server component main package for Varasto
package stoserver

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/gokit/systemdinstaller"
	"github.com/function61/varasto/pkg/logtee"
	"github.com/function61/varasto/pkg/restartcontroller"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stothumbserver"
	"github.com/spf13/cobra"
)

func serverMain() error {
	logTail := logtee.NewStringTail(50)

	// writes to upstream all end up in the sink, but logTail.Snapshot() only
	// returns the last "capacity" lines
	rootLogger := logex.StandardLoggerTo(logtee.NewLineSplitterTee(os.Stderr, func(line string) {
		logTail.Write(line)
	}))

	ctx, cancel := context.WithCancel(context.Background())

	go func(logger *log.Logger) {
		logex.Levels(logger).Info.Printf(
			"Got %s; stopping",
			<-ossignal.InterruptOrTerminate())

		cancel()
	}(logex.Prefix("main", rootLogger))

	restartable := restartcontroller.New(logex.Prefix("restartcontroller", rootLogger))

	return restartable.Run(ctx, func(ctx context.Context) error {
		// a short wrapper to adapt context-based cancellation to stopper-based cancellation
		workers := stopper.NewManager()
		go func() {
			<-ctx.Done()
			workers.StopAllWorkersAndWait()
		}()

		return runServer(rootLogger, logTail, workers.Stopper(), restartable)
	})
}

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the server component",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			exitIfError(serverMain())
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "dbimport",
		Short: "Imports database",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			scf, err := readServerConfigFile()
			exitIfError(err)

			exitIfError(stodbimportexport.Import(os.Stdin, scf.DbLocation))
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

			exitIfError(systemdinstaller.Install(serviceFile))

			fmt.Println(systemdinstaller.GetHints(serviceFile))
		},
	})

	cmd.AddCommand(stothumbserver.Entrypoint())

	return cmd
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
