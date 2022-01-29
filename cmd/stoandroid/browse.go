package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/function61/gokit/fileexists"
)

func browsePage(winCtx *winContext) {
	buttons := []fyne.CanvasObject{}

	for _, ld := range localDirs {
		ld := ld // pin

		btn := widget.NewButton(ld.Name(), func() {
			browsePageSelect(winCtx, ld)
		})

		buttons = append(buttons, btn)
	}

	winCtx.win.SetContent(vertically(
		createHeading("Browse"),
		vertically(buttons...),
		mkBackButton(winCtx)))
}

func browsePageSelect(winCtx *winContext, sel LocalDir) {
	duplicateScanBtn := widget.NewButton("Duplicate scan", func() {
		duplicateScanner(winCtx, duplicateScannerLocation{
			name: sel.Name(),
			dir:  sel.Path,
		})
	})

	subdirItems := []fyne.CanvasObject{}
	if sel.enableDiscoverChildren {
		dentries, err := ioutil.ReadDir(sel.Path)
		if err != nil {
			dialog.ShowError(err, winCtx.win)
			return
		}

		for _, dentry := range dentries {
			if !dentry.IsDir() {
				continue
			}

			// TODO: assuming direct descendants are repos
			subDir := repoDir(filepath.Join(sel.Path, dentry.Name()), dentry.Name())

			goToSubdirBtn := widget.NewButton(subDir.Name(), func() {
				browsePageSelect(winCtx, subDir)
			})

			subdirItems = append(subdirItems, goToSubdirBtn)
		}
	}

	// TODO: don't hardcode this
	isWorkdir, err := fileexists.Exists(filepath.Join(sel.Path, ".varasto"))
	if err != nil {
		dialog.ShowError(err, winCtx.win)
		return
	}

	isWorkdirLabel := widget.NewLabel(fmt.Sprintf("Is workdir: %v", isWorkdir))

	pushBtn := widget.NewButton("Push", func() {
		pushPage(winCtx, sel.Path)
	})
	setEnabled(&pushBtn.DisableableWidget, sel.enablePush)

	sortPhotosBtn := widget.NewButton("Sort photos", func() {
		mvuPhotosPage(winCtx, sel)
	})
	setEnabled(&sortPhotosBtn.DisableableWidget, sel.enableMover)

	winCtx.win.SetContent(vertically(
		createHeading(sel.Name()),
		widget.NewSeparator(),
		isWorkdirLabel,
		widget.NewSeparator(),
		vertically(subdirItems...),
		pushBtn,
		sortPhotosBtn,
		duplicateScanBtn,
		mkBackButton(winCtx)))
}

/*
	widget.NewButton("Push 2021-01", func() {
		pushPage(winCtx, "/storage/emulated/0/DCIM/Camera/2021-01 - Unsorted")
	}),
	widget.NewButton("Clone 2020-12", func() {
		clonePage(winCtx, "/storage/emulated/0/DCIM/Camera", "LCwtN8F2BLA") // LCwtN8F2BLA = "2020-12 - Unsorted"
	}),
	widget.NewButton("mvu photos", func() {
		mvuPhotosPage(winCtx)
	}),
	widget.NewButton("Upload VID_20190310_134312.mp4", func() {
		uploadPage(winCtx, "/storage/emulated/0/DCIM/Camera/VID_20190310_134312.mp4")
	}),
		widget.NewButton("Upload 2", func() {
			uploadPage(winCtx, "/storage/emulated/0/DCIM/Camera/VID_20190310_140702.mp4")
		}),
	duplicateScannerTarget(camera()),
*/
