// +build !windows

package varastofuse

import (
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/stopper"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/varastofuse/vstofusetypes"
	"github.com/gorilla/mux"
	"net/http"
)

type handlers struct {
	sigs *sigFabric
}

func (h *handlers) FuseMount(rctx *httpauth.RequestContext, pars vstofusetypes.CollectionId, w http.ResponseWriter, r *http.Request) {
	h.sigs.mount <- pars.Id
}

func (h *handlers) FuseUnmount(rctx *httpauth.RequestContext, pars vstofusetypes.CollectionId, w http.ResponseWriter, r *http.Request) {
	h.sigs.unmount <- pars.Id
}

func rpcServe(sigs *sigFabric, stop *stopper.Stopper) {
	router := mux.NewRouter()

	srv := http.Server{
		Addr:    ":8689",
		Handler: router,
	}

	var han vstofusetypes.HttpHandlers = &handlers{sigs}

	vstofusetypes.RegisterRoutes(han, createDummyMiddlewares(), muxregistrator.New(router))

	go func() {
		defer stop.Done()

		<-stop.Signal

		if err := srv.Shutdown(nil); err != nil {
			panic(err)
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		panic(err)
	}
}

func createDummyMiddlewares() httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
	}
}
