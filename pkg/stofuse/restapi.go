package stofuse

import (
	"context"
	"net/http"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/varasto/pkg/gokitbp"
	"github.com/function61/varasto/pkg/stofuse/stofusetypes"
	"github.com/function61/varasto/pkg/stoutils"
)

type handlers struct {
	sigs *sigFabric
}

func (h *handlers) FuseUnmountAll(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	h.sigs.unmountAll <- nil
}

func rpcStart(addr string, sigs *sigFabric, tasks *taskrunner.Runner) error {
	router := http.NewServeMux()

	var han stofusetypes.HttpHandlers = &handlers{sigs}

	stofusetypes.RegisterRoutes(han, createDummyMiddlewares(), func(method, path string, fn http.HandlerFunc) {
		router.HandleFunc(method+" "+path, fn)
	})

	listener, err := stoutils.CreateTCPOrDomainSocketListener(addr, logex.Levels(logex.Discard))
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: gokitbp.DefaultReadHeaderTimeout,
	}

	tasks.Start("rpc "+addr, func(_ context.Context) error {
		return httputils.RemoveGracefulServerClosedError(srv.Serve(listener))
	})

	tasks.Start("rpcshutdowner", httputils.ServerShutdownTask(srv))

	return nil
}

func createDummyMiddlewares() httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
	}
}
