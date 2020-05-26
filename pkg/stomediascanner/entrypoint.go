package stomediascanner

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/taskrunner"
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

	listener, err := stoutils.CreateTcpOrDomainSocketListener(addr, logl)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler: router,
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

			ctx, cancel := context.WithCancel(ossignal.InterruptOrTerminateBackgroundCtx(rootLogger))

			go func() {
				// wait for stdin EOF (or otherwise broken pipe)
				_, _ = io.Copy(ioutil.Discard, os.Stdin)

				logex.Levels(rootLogger).Error.Println("parent process died (detected by closed stdin) - stopping")

				cancel()
			}()

			if err := logic(ctx, addr, rootLogger); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
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
