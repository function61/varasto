package main

import (
	"fyne.io/fyne/widget"
)

func settingsPage(winCtx *winContext) {
	form := widget.NewForm(
		widget.NewFormItem("Server address", widget.NewLabel(getClient().ServerAddr)))

	winCtx.win.SetContent(vertically(
		createHeading("Settings"),
		form,
		mkBackButton(winCtx)))
}
