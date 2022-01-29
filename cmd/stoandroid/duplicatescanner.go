package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoclient"
)

type duplicateScannerLocation struct {
	name string
	dir  string
}

func (d duplicateScannerLocation) Title() string {
	return fmt.Sprintf("Duplicate scanner: %s", d.name)
}

func camera() duplicateScannerLocation {
	return duplicateScannerLocation{
		name: "Camera",
		// dir:  "/storage/emulated/0/DCIM/Camera",
		dir: "/storage/self/primary/DCIM/Camera",
	}
}

func camScanner() duplicateScannerLocation {
	return duplicateScannerLocation{
		name: "CamScanner",
		dir:  "/storage/emulated/0/CamScanner",
	}
}

/*
func allLocations() []duplicateScannerLocation {
	return []duplicateScannerLocation{
		camera(),
	}
}
*/

func duplicateScanner(winCtx *winContext, loc duplicateScannerLocation) {
	backButton := widget.NewButton("Back", func() {
		toHomePage(winCtx)
	})

	status := widget.NewLabel("Getting file list from Varasto")
	status.Wrapping = fyne.TextWrapWord

	leftToScan := widget.NewLabel("...")
	duplicate := widget.NewLabel("...")
	newFiles := widget.NewLabel("...")

	progressBar := widget.NewProgressBar()

	// TODO: add scan speed
	form := widget.NewForm(
		widget.NewFormItem("Status", status),
		widget.NewFormItem("Left to scan", leftToScan),
		widget.NewFormItem("Duplicate", duplicate),
		widget.NewFormItem("New files", newFiles),
		widget.NewFormItem("Progress", progressBar),
		widget.NewFormItem("", backButton))

	winCtx.win.SetContent(vertically(
		createHeading(loc.Title()),
		form))

	progressFn := func(stats *duplicateScanReport) {
		status.SetText("Scanning...")

		leftToScan.SetText(strconv.Itoa(stats.leftToScan))
		duplicate.SetText(strconv.Itoa(len(stats.duplicate)))
		newFiles.SetText(strconv.Itoa(stats.newFiles))

		progressBar.Value = stats.Progress()
		progressBar.Refresh()
	}

	report, err := scanForDuplicates(winCtx.ctx, loc, progressFn)
	if err != nil {
		status.SetText(err.Error())
		return
	}

	displayScanReport(winCtx, report)
}

func scanForDuplicates(
	ctx context.Context,
	loc duplicateScannerLocation,
	progress func(stats *duplicateScanReport),
) (*duplicateScanReport, error) {
	conf := getClient()

	// TODO: move this below hashes from server (here for faster debugging)
	dentries, err := func() ([]os.FileInfo, error) {
		allEntries, err := ioutil.ReadDir(loc.dir)
		if err != nil {
			return nil, err
		}

		files := []os.FileInfo{}
		for _, entry := range allEntries {
			if entry.IsDir() {
				continue
			}

			files = append(files, entry)
		}

		return files, nil
	}()
	if err != nil {
		return nil, err
	}

	hashesForFilesThatServerHas, err := getHashesFromServer(ctx, conf)
	if err != nil {
		return nil, err
	}

	stats := &duplicateScanReport{
		duplicate:  []string{},
		leftToScan: len(dentries),
	}

	checkDuplicateOneFile := func(fileInfo os.FileInfo) error {
		filePath := filepath.Join(loc.dir, fileInfo.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		fileHash := sha256.New()
		if _, err := io.Copy(fileHash, file); err != nil {
			return err
		}

		fileHashHex := fmt.Sprintf("%x", fileHash.Sum(nil))

		_, duplicate := hashesForFilesThatServerHas[fileHashHex]

		stats.leftToScan--
		if duplicate {
			stats.duplicate = append(stats.duplicate, filePath)
		} else {
			stats.newFiles++
		}

		return nil
	}

	for _, dentry := range dentries {
		// TODO: parallelize
		if err := checkDuplicateOneFile(dentry); err != nil {
			return nil, err
		}

		progress(stats)
	}

	return stats, nil
}

func displayScanReport(winCtx *winContext, report *duplicateScanReport) {
	backButton := widget.NewButton("Back", func() {
		toHomePage(winCtx)
	})

	progressBar := widget.NewProgressBar()

	progressFn := func(progress float64) {
		progressBar.Value = progress
		progressBar.Refresh()
	}

	removeDuplicatesBtn := widget.NewButton("Remove duplicates", func() {
		if err := removeDuplicates(report, progressFn); err != nil {
			dialog.ShowError(err, winCtx.win)
		}
	})
	removeDuplicatesBtn.Style = widget.PrimaryButton

	form := widget.NewForm(
		widget.NewFormItem("Duplicate", widget.NewLabel(strconv.Itoa(len(report.duplicate)))),
		widget.NewFormItem("New files", widget.NewLabel(strconv.Itoa(report.newFiles))),
		widget.NewFormItem("Progress", progressBar),
		widget.NewFormItem("", horizontally(backButton, removeDuplicatesBtn)))

	winCtx.win.SetContent(vertically(
		createHeading("Duplicate scan report"),
		form))
}

func removeDuplicates(report *duplicateScanReport, progress func(progress float64)) error {
	total := float64(len(report.duplicate))

	for idx, duplicate := range report.duplicate {
		if err := os.Remove(duplicate); err != nil {
			return err
		}

		progress(float64(idx+1) / total)
	}

	return nil
}

// TODO: this is duplicated in stodupremover
func getHashesFromServer(ctx context.Context, conf *stoclient.ClientConfig) (map[string]bool, error) {
	res, err := ezhttp.Get(
		ctx,
		conf.UrlBuilder().DatabaseExportSha256s(),
		ezhttp.AuthBearer(conf.AuthToken),
		ezhttp.Client(conf.HttpClient()))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	hashes := map[string]bool{}

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// lines are formatted "<sha256lowercasehex> <filename>"
			hashes[line[0:256/8*2]] = true
		}
	}

	return hashes, scanner.Err()
}

type duplicateScanReport struct {
	leftToScan int
	duplicate  []string
	newFiles   int
}

func (s *duplicateScanReport) Progress() float64 {
	done := len(s.duplicate) + s.newFiles
	total := s.leftToScan + done

	return float64(done) / float64(total)
}

func getClient() *stoclient.ClientConfig {
	panic("AuthToken redacted")
	return &stoclient.ClientConfig{
		ServerAddr: "https://varasto.example.net",
		AuthToken:  "...",
	}
}
