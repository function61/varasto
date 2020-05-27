package stomediascanner

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stomediascanner/stomediascantypes"
	"github.com/gorilla/mux"
)

type Controller struct {
	clientConf *stoclient.ClientConfig
	logl       *logex.Leveled
}

func NewController(
	router *mux.Router,
	mwares httpauth.MiddlewareChainMap,
	logger *log.Logger,
) (*Controller, error) {
	clientConf, err := stoclient.ReadConfig()
	if err != nil {
		return nil, err
	}

	c := &Controller{
		clientConf: clientConf,
		logl:       logex.Levels(logger),
	}

	stomediascantypes.RegisterRoutes(c, mwares, muxregistrator.New(router))

	return c, nil
}

func (c *Controller) TriggerScan(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	collectionId := mux.Vars(r)["id"]
	mode := r.URL.Query().Get("mode")

	// a = "allow destructive changes"
	moveNamedThumbnails := mode == "a"

	if err := collectionThumbnails(
		r.Context(),
		collectionId,
		moveNamedThumbnails,
		c.clientConf,
		c.logl,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (c *Controller) Task() func(context.Context) error {
	return func(ctx context.Context) error {
		return c.runTask(ctx)
	}
}

func (c *Controller) runTask(ctx context.Context) error {
	// give the server time to start (TODO: ugly)
	time.Sleep(2 * time.Second)

	state, err := discoverState(ctx, c.clientConf)
	if err != nil {
		return err
	}

	// so we can detect if we need to save
	serverState := state

	c.logl.Debug.Printf("start state at: %s", state)

	for {
		changefeed, err := discoverChanges(ctx, state, c.clientConf)
		if err != nil {
			return err
		}

		for _, item := range changefeed {
			if err := collectionThumbnails(ctx, item.CollectionId, false, c.clientConf, c.logl); err != nil {
				return err
			}

			// move state forward
			state = item.Cursor
		}

		select {
		case <-ctx.Done():
			if state != serverState {
				c.logl.Info.Printf("state advanced to %s (from %s); saving", state, serverState)

				// need background ctx because "ctx" is canceled
				if err := setState(context.Background(), state, c.clientConf); err != nil {
					return err
				}
			}

			return nil
		case <-time.After(5 * time.Second):
			c.logl.Debug.Printf("polling; last time processed (%d) changefeed items\n", len(changefeed))
		}
	}
}
