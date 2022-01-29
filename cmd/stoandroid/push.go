package main

import (
	"strings"

	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"github.com/function61/varasto/pkg/stoclient"
)

func pushPage(winCtx *winContext, dir string) {
	client := getClient()

	stoWorkdir, err := client.NewMaybeWorkdirLocation(dir)
	if err != nil {
		dialog.ShowError(err, winCtx.win)
		return
	}
	isWorkdir := stoWorkdir != nil

	if !isWorkdir { // requires adoption
		showAdoptionPage(winCtx, dir, client)
		return
	}

	accordionItems := []*widget.AccordionItem{}

	ch, err := stoclient.ComputeChangeset(winCtx.ctx, stoWorkdir, stoclient.NewBlobDiscoveredNoopListener())
	if err != nil {
		dialog.ShowError(err, winCtx.win)
		return
	}

	if len(ch.FilesCreated) > 0 {
		names := []string{}
		for _, f := range ch.FilesCreated {
			names = append(names, f.Path)
		}

		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Created",
			Detail: widget.NewLabel(strings.Join(names, "\n")),
			Open:   true,
		})
	}

	if len(ch.FilesUpdated) > 0 {
		names := []string{}
		for _, f := range ch.FilesUpdated {
			names = append(names, f.Path)
		}

		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Updated",
			Detail: widget.NewLabel(strings.Join(names, "\n")),
			Open:   true,
		})
	}

	if len(ch.FilesDeleted) > 0 {
		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Deleted",
			Detail: widget.NewLabel(strings.Join(ch.FilesDeleted, "\n")),
			Open:   true,
		})
	}

	if !ch.AnyChanges() {
		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Nothing to do",
			Detail: widget.NewLabel("No changes"),
			Open:   true,
		})
	}

	// initially empty, rows will get added dynamically
	uploadProgressItems := fyne.NewContainerWithLayout(
		layout.NewFormLayout())

	var pushBtn *widget.Button
	pushBtn = widget.NewButton("Push", func() {
		pushBtn.Disable()

		progressListener := stoclient.NewUploadProgressCustomUI(func(objects []*stoclient.ObjectUploadStatus) error {
			clearContainer(uploadProgressItems)

			for _, obj := range objects {
				uploadProgressItems.Add(widget.NewLabel(obj.Key))
				pbar := widget.NewProgressBar()
				pbar.Value = obj.Progress()
				uploadProgressItems.Add(pbar)
			}

			uploadProgressItems.Refresh()

			return nil
		})

		if err := stoclient.Push(winCtx.ctx, stoWorkdir, progressListener); err != nil {
			dialog.ShowError(err, winCtx.win)
			pushBtn.Enable()
			return
		}

		winCtx.app.SendNotification(fyne.NewNotification(
			"Success",
			"Collection committed successfully"))

		/*
			ch, err := stoclient.ComputeChangeset(winCtx.ctx, stoWorkdir, stoclient.NewBlobDiscoveredNoopListener())
			if err != nil {
				return err
			}
		*/
	})
	pushBtn.Style = widget.PrimaryButton
	if !ch.AnyChanges() {
		pushBtn.Disable()
	}

	backBtn := widget.NewButton("Back", func() {
		toHomePage(winCtx)
	})

	winCtx.win.SetContent(vertically(
		createHeading("Push"),
		widget.NewAccordion(accordionItems...),
		uploadProgressItems,
		horizontally(
			backBtn,
			pushBtn,
		)))
}

func clearContainer(container *fyne.Container) {
	for _, object := range container.Objects {
		container.Remove(object)
	}
}

func showAdoptionPage(winCtx *winContext, dir string, client *stoclient.ClientConfig) {
	winCtx.win.SetContent(createHeading("adoption pending"))

	dialog.ShowEntryDialog("Adoption", "Collection id", func(parentDirectoryId string) {
		if parentDirectoryId == "" {
			return
		}

		if err := client.Adopt(winCtx.ctx, dir, parentDirectoryId); err != nil {
			dialog.ShowError(err, winCtx.win)
			return
		}

		winCtx.win.SetContent(vertically(
			createHeading("Adoption done"),
			widget.NewSeparator(),
			mkBackButton(winCtx),
		))
	}, winCtx.win)
}
