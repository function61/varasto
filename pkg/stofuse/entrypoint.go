// FUSE adapter for interfacing with Varasto from filesystem
package stofuse

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fuse",
		Short: "Varasto-FUSE integration",
	}

	addr := ":8689"
	unmountFirst := false
	stopIfStdinCloses := false

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Mounts a FUSE-based FS to serve collections from Varasto",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			ctx, cancel := context.WithCancel(osutil.CancelOnInterruptOrTerminate(
				rootLogger))
			defer cancel()

			if stopIfStdinCloses {
				registerStdinCloseAsCancellationSignal(cancel, rootLogger)
			}

			osutil.ExitIfError(serve(ctx, addr, unmountFirst, rootLogger))
		},
	}

	serveCmd.Flags().StringVarP(&addr, "addr", "", addr, "Address to listen on")
	serveCmd.Flags().BoolVarP(&unmountFirst, "unmount-first", "u", unmountFirst, "Umount the mount-path first (maybe unclean shutdown previously)")
	serveCmd.Flags().BoolVarP(&stopIfStdinCloses, "stop-if-stdin-closes", "", stopIfStdinCloses, "Stop the server if stdin closes (= detect if parent process dies)")

	cmd.AddCommand(serveCmd)

	return cmd
}

func serve(ctx context.Context, addr string, unmountFirst bool, logger *log.Logger) error {
	logl := logex.Levels(logger)

	conf, err := stoclient.ReadConfig()
	if err != nil {
		return err
	}

	// connects RPC API and FUSE server together
	sigs := newSigs()

	tasks := taskrunner.New(ctx, logger)

	// do this before starting other tasks, because we're not cancelling the tasks that
	// come after rpcStart()
	if err := rpcStart(addr, sigs, tasks); err != nil {
		return err
	}

	tasks.Start("fusesrv", func(ctx context.Context) error {
		return fuseServe(ctx, sigs, *conf, unmountFirst, logl)
	})

	return tasks.Wait()
}

func registerStdinCloseAsCancellationSignal(cancel context.CancelFunc, logger *log.Logger) {
	go func() {
		// wait for stdin EOF (or otherwise broken pipe)
		_, _ = io.Copy(ioutil.Discard, os.Stdin)

		logex.Levels(logger).Error.Println(
			"parent process died (detected by closed stdin) - stopping")

		cancel()
	}()
}
