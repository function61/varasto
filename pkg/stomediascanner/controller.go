package stomediascanner

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stomediascanner/stomediascantypes"
)

type Controller struct {
	clientConf *stoclient.ClientConfig
	logl       *logex.Leveled
}

func NewController(
	router *http.ServeMux,
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

	stomediascantypes.RegisterRoutes(c, mwares, func(method, path string, fn http.HandlerFunc) {
		router.HandleFunc(method+" "+path, fn)
	})

	return c, nil
}

func (c *Controller) TriggerScan(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	collectionID := r.PathValue("id")
	mode := r.URL.Query().Get("mode")

	// a = "allow destructive changes"
	moveNamedThumbnails := mode == "a"

	if err := collectionThumbnails(
		r.Context(),
		collectionID,
		moveNamedThumbnails,
		c.clientConf,
		c.logl,
	); err != nil {
		http.Error(w, fmt.Errorf("collectionThumbnails: %w", err).Error(), http.StatusInternalServerError)
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
		return fmt.Errorf("discoverState: %w", err)
	}

	// so we can detect if we need to save
	serverState := state

	c.logl.Debug.Printf("start state at: %s", state)

	// save progress to server on this interval. it doesn't matter much if we lose any progress
	// tracking, since we'd just revisit some already processed collections and we'd just
	// discover that there's no more work to do
	updateServerStateInterval := time.Tick(5 * time.Second)

	updateServerState := func() error {
		if serverState == state { // nothing to do
			return nil
		}

		c.logl.Info.Printf("state advanced to %s (from %s); saving to server", state, serverState)

		if err := setState(ctx, state, c.clientConf); err != nil {
			return err
		}

		serverState = state

		return nil
	}

	for {
		changefeed, err := discoverChanges(ctx, state, c.clientConf)
		if err != nil {
			return fmt.Errorf("discoverChanges: %w", err)
		}

		for _, item := range changefeed {
			if err := collectionThumbnails(ctx, item.CollectionId, false, c.clientConf, c.logl); err != nil {
				return fmt.Errorf("collectionThumbnails: %w", err)
			}

			// move state forward
			state = item.Cursor

			select {
			case <-updateServerStateInterval:
				if err := updateServerState(); err != nil {
					return err
				}
			default:
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-updateServerStateInterval:
			if err := updateServerState(); err != nil {
				return err
			}
		case <-time.After(3 * time.Second): // sleep for a while as not to hammer the server
			c.logl.Debug.Printf("polling; last time processed (%d) changefeed items\n", len(changefeed))
		}
	}
}
