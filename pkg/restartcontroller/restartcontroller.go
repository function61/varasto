// Wrapper for running a restartable fn. it gets its restart signal via context cancellation
package restartcontroller

import (
	"context"
	"errors"
	"github.com/function61/gokit/logex"
	"log"
)

type Controller struct {
	restart chan interface{}
	logl    *logex.Leveled
}

func New(logger *log.Logger) *Controller {
	return &Controller{
		restart: make(chan interface{}),
		logl:    logex.Levels(logger),
	}
}

// returns immediately
func (r *Controller) Restart() error {
	select {
	case r.restart <- nil:
		return nil
	default:
		return errors.New("unable to send restart signal - runner busy or exited?")
	}
}

func (r *Controller) Run(ctx context.Context, run func(ctx context.Context) error) error {
	stopped := make(chan error)

	var cancelFn context.CancelFunc

	start := func() {
		var subCtx context.Context
		subCtx, cancelFn = context.WithCancel(ctx)

		go func() {
			stopped <- run(subCtx)
		}()
	}

	start()

	for {
		select {
		case <-r.restart:
			r.logl.Info.Println("stopping due to restart request")

			cancelFn()

			if err := <-stopped; err != nil {
				r.logl.Error.Printf("stopped but with error (will start anyway): %v", err)
			} else {
				r.logl.Info.Println("graceful stop; starting")
			}

			start()
		case err := <-stopped:
			return err
		}
	}
}
