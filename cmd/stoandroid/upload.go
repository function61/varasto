package main

import (
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/function61/varasto/pkg/byteshuman"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
)

func uploadPage(winCtx *winContext, filePath string) {
	clientConfig := getClient()
	client := clientConfig.Client()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		dialog.ShowError(err, winCtx.win)
		return
	}

	fileRelative := filepath.Base(filePath)

	collInput := widget.NewEntry()

	deleteAfterUpload := widget.NewCheck("Delete after upload", nil)

	progressBar := widget.NewProgressBar()

	form := widget.NewForm(
		widget.NewFormItem("Name", widget.NewLabel(fileRelative)),
		widget.NewFormItem("Size", widget.NewLabel(byteshuman.Humanize(uint64(fileInfo.Size())))),
		widget.NewFormItem("Collection", collInput), // TODO: show collection name
		widget.NewFormItem("", deleteAfterUpload),
		widget.NewFormItem("Progress", progressBar))

	doUpload := func() error {
		coll, err := client.FetchCollectionMetadata(winCtx.ctx, collInput.Text)
		if err != nil {
			return err
		}

		progressListener := stoclient.NewUploadProgressCustomUI(func(objects []*stoclient.ObjectUploadStatus) error {
			if len(objects) > 0 {
				progressBar.Value = objects[0].Progress()
				progressBar.Refresh()
			}

			return nil
		})

		buploader := stoclient.NewBackgroundUploader(
			winCtx.ctx,
			// stoclient.BackgroundUploaderConcurrency,
			1, // TODO: hack around hitting time limit
			*clientConfig,
			progressListener)

		file, err := stoclient.ScanFileAndDiscoverBlobs(winCtx.ctx, filePath, fileRelative, fileInfo, coll.ID, buploader)
		if err != nil {
			return err
		}

		if err := buploader.WaitFinished(); err != nil {
			return err
		}

		changeset := stotypes.NewChangeset(
			stoutils.NewCollectionChangesetId(),
			coll.Head,
			time.Now(),
			[]stotypes.File{*file},
			[]stotypes.File{},
			[]string{})

		_, err = client.Commit(winCtx.ctx, coll.ID, changeset)
		return err
	}

	var uploadBtn *widget.Button
	uploadBtn = widget.NewButton("Upload", func() {
		uploadBtn.Disable()
		deleteAfterUpload.Disable()

		if err := doUpload(); err != nil {
			dialog.ShowError(err, winCtx.win)

			uploadBtn.Enable()
			deleteAfterUpload.Enable()
			return
		}

		dialog.ShowInformation("Done", "Uploaded successfully", winCtx.win)

		if deleteAfterUpload.Checked {
			if err := os.Remove(filePath); err != nil {
				dialog.ShowError(err, winCtx.win)
			}
		}
	})
	uploadBtn.Style = widget.PrimaryButton

	winCtx.win.SetContent(vertically(
		createHeading("Upload"),
		form,
		horizontally(
			widget.NewButton("Back", func() {
				toHomePage(winCtx)
			})),
		uploadBtn))
}
