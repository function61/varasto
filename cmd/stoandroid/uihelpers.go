package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

func vertically(els ...fyne.CanvasObject) *fyne.Container {
	return fyne.NewContainerWithLayout(
		layout.NewVBoxLayout(),
		els...)
}

func horizontally(els ...fyne.CanvasObject) *fyne.Container {
	return fyne.NewContainerWithLayout(
		layout.NewHBoxLayout(),
		els...)
}

func mkBackButton(winCtx *winContext) *widget.Button {
	return widget.NewButton("Back", func() {
		toHomePage(winCtx)
	})
}

func setEnabled(wid *widget.DisableableWidget, enable bool) {
	if enable {
		wid.Enable()
	} else {
		wid.Disable()
	}
}
