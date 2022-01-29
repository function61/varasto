package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/function61/varasto/pkg/stomvu"
)

func mvuPhotosPage(winCtx *winContext, dirInfo LocalDir) {
	// works
	// dir := "/sdcard/data/colornote/backup"
	// dir := "/storage/emulated/0/data/colornote/backup"
	// dir := camera().dir

	plan, err := stomvu.ComputePlan(dirInfo.Path, stomvu.PhotoOrVideoDateFromFilename)
	if err != nil {
		dialog.ShowError(err, winCtx.win)
		return
	}

	accordionItems := []*widget.AccordionItem{}

	for _, fileTargets := range plan.FileTargets {
		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  fileTargets.Target,
			Detail: widget.NewLabel(strings.Join(fileTargets.Sources, "\n")),
		})
	}

	if len(plan.DirectoryTargets) > 0 {
		dirExplanations := []string{}
		for _, directoryTargets := range plan.DirectoryTargets {
			// len(Sources) always is 1
			dirExplanations = append(dirExplanations, fmt.Sprintf("%s => %s", strings.Join(directoryTargets.Sources, ", "), directoryTargets.Target))
		}

		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Directories",
			Detail: widget.NewLabel(strings.Join(dirExplanations, "\n")),
		})
	}

	if len(plan.Dunno) > 0 {
		dunnoItemsAndExplanation := append([]string{
			"(already renamed or didn't match renaming criteria)",
			"",
		}, plan.Dunno...)

		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Skipped",
			Detail: widget.NewLabel(strings.Join(dunnoItemsAndExplanation, "\n")),
		})
	}

	nothingToDo := (len(plan.FileTargets) + len(plan.DirectoryTargets)) == 0

	if nothingToDo {
		accordionItems = append(accordionItems, &widget.AccordionItem{
			Title:  "Nothing to do",
			Detail: widget.NewLabel("Nothing to move was found"),
			Open:   true,
		})
	}

	moveBtn := widget.NewButton("Move", func() {
		if err := stomvu.ExecutePlan(plan); err != nil {
			dialog.ShowError(err, winCtx.win)
			return
		}

		winCtx.app.SendNotification(fyne.NewNotification(
			"Success",
			fmt.Sprintf("%d item(s) moved successfully", plan.NumSources())))

		mvuPhotosPage(winCtx, dirInfo) // go back
	})
	moveBtn.Style = widget.PrimaryButton
	if nothingToDo {
		moveBtn.Disable()
	}

	backBtn := widget.NewButton("Back", func() {
		toHomePage(winCtx)
	})

	winCtx.win.SetContent(vertically(
		createHeading("Photo & video mover"),
		widget.NewAccordion(accordionItems...),
		horizontally(
			backBtn,
			moveBtn,
		)))
}
