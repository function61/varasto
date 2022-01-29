package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/widget"
)

func clonePage(winCtx *winContext, dir string, collId string) {
	client := getClient()

	statusLabel := widget.NewLabel("Cloning")

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- client.Clone(
			winCtx.ctx,
			collId,
			"",  // => use head revision
			dir, // parent dir
			"",  // => use name from server
		)
	}()

	go func() {
		tick := time.NewTicker(1 * time.Second)
		started := time.Now()
		for {
			select {
			case <-tick.C:
				statusLabel.SetText(fmt.Sprintf("%s elapsed", time.Since(started)))
			case err := <-resultCh:
				if err != nil {
					statusLabel.SetText("Done, error")
				} else {
					statusLabel.SetText("Done, success")
				}
				return
			}
		}
	}()

	winCtx.win.SetContent(vertically(
		createHeading("Clone"),
		statusLabel,
		mkBackButton(winCtx),
	))
}
