package stofuse

import (
	"context"
	"net/http"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/stofuse/stofusetypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/gorilla/mux"
)

type handlers struct {
	sigs *sigFabric
}

func (h *handlers) FuseUnmountAll(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	h.sigs.unmountAll <- nil
}

func rpcServe(addr string, sigs *sigFabric, stop *stopper.Stopper) error {
	router := mux.NewRouter()

	var han stofusetypes.HttpHandlers = &handlers{sigs}

	stofusetypes.RegisterRoutes(han, createDummyMiddlewares(), muxregistrator.New(router))

	listener, err := stoutils.CreateTcpOrDomainSocketListener(addr, logex.Levels(logex.Discard))
	if err != nil {
		return err
	}

	srv := http.Server{
		Handler: router,
	}

	go func() {
		defer stop.Done()

		<-stop.Signal

		if err := srv.Shutdown(context.TODO()); err != nil {
			panic(err)
		}
	}()

	if err := srv.Serve(listener); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func createDummyMiddlewares() httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
	}
}
