package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/smart"
)

func main() {
	exitIfError(logic(
		osutil.CancelOnInterruptOrTerminate(nil)))
}

func logic(ctx context.Context) error {
	routes := httputils.NewMethodMux()
	routes.GET.HandleFunc("/smart", func(w http.ResponseWriter, r *http.Request) {
		device := r.URL.Query().Get("device")
		deviceType := r.URL.Query().Get("devicetype")

		if device == "" { // some drives e.g. ones behind QNAP TR-004 require special switches
			http.Error(w, "device cannot be empty", http.StatusBadRequest)
			return
		}

		cmd := []string{"smartctl", "--json", "--all", device}

		// --device=jmb39x-q,3
		if deviceType != "" {
			cmd = append(cmd, fmt.Sprintf("--device=%s", deviceType))
		}

		// buffer in case '$ smartctl' exec fails, so we can still respond with HTTP error
		smartctlJSONOutput := &bytes.Buffer{}
		smartctlStderr := &bytes.Buffer{}

		smartctl := exec.CommandContext(r.Context(), cmd[0], cmd[1:]...)
		smartctl.Stdout = smartctlJSONOutput
		smartctl.Stderr = smartctlStderr

		if err := smart.SilenceSmartCtlAutomationHostileErrors(smartctl.Run()); err != nil {
			errWrapped := fmt.Sprintf(
				"smartctl: %v: stdout=%s stderr=%s",
				err,
				smartctlJSONOutput.String(),
				smartctlStderr.String())

			http.Error(w, errWrapped, http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(smartctlJSONOutput.Bytes())
		}
	})

	srv := &http.Server{
		Addr:    ":80",
		Handler: routes,
	}

	return cancelableServer(ctx, srv, srv.ListenAndServe)
}

// helper for adapting context cancellation to shutdown the HTTP listener
func cancelableServer(ctx context.Context, srv *http.Server, listener func() error) error {
	shutdownerCtx, cancel := context.WithCancel(ctx)

	shutdownResult := make(chan error, 1)

	// this is the actual shutdowner
	go func() {
		// triggered by parent cancellation
		// (or below for cleanup if ListenAndServe() failed by itself)
		<-shutdownerCtx.Done()

		// can't use parent ctx b/c it'd cancel the Shutdown() itself
		shutdownResult <- srv.Shutdown(context.Background())
	}()

	err := listener()

	// ask shutdowner to stop. this is useful only for cleanup where listener failed before
	// it was requested to shut down b/c parent cancellation didn't happen and thus the
	// shutdowner would still wait.
	cancel()

	if err == http.ErrServerClosed { // expected for graceful shutdown (not actually error)
		return <-shutdownResult // should be nil, unless shutdown fails
	} else {
		// some other error
		// (or nil, but http server should always exit with non-nil error)
		return err
	}
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
