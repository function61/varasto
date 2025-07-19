package stomediascanner

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/varasto/pkg/gokitbp"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

const (
	Verb = "mediascanner"
)

func logic(ctx context.Context, addr string, rootLogger *log.Logger) error {
	router := mux.NewRouter()
	logl := logex.Levels(rootLogger)

	thumbController, err := NewController(router, createDummyMiddlewares(), rootLogger)
	if err != nil {
		return err
	}

	listener, err := stoutils.CreateTCPOrDomainSocketListener(addr, logl)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: gokitbp.DefaultReadHeaderTimeout,
	}

	tasks := taskrunner.New(ctx, rootLogger)

	tasks.Start("task", thumbController.Task())

	tasks.Start("listener "+listener.Addr().String(), func(ctx context.Context) error { return httputils.RemoveGracefulServerClosedError(srv.Serve(listener)) })

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}

func Entrypoint() *cobra.Command {
	addr := ":8688"

	cmd := &cobra.Command{
		Use:   Verb,
		Short: "Starts the media scanner server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			ctx, cancel := context.WithCancel(osutil.CancelOnInterruptOrTerminate(rootLogger))

			go func() {
				// wait for stdin EOF (or otherwise broken pipe)
				_, _ = io.Copy(io.Discard, os.Stdin)

				logex.Levels(rootLogger).Error.Println("parent process died (detected by closed stdin) - stopping")

				cancel()
			}()

			osutil.ExitIfError(logic(ctx, addr, rootLogger))
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "", addr, "Address to listen on")

	return cmd
}

func createDummyMiddlewares() httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
	}
}
