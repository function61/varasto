package main

import (
	"context"
	"net/url"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

// NOTE: targetSdkVersion=28
// https://stackoverflow.com/a/56809469

// compile in project root with
// $ fyne-cross android -app-id=com.function61.varasto -icon=misc/varasto-logo/varasto-mobile.png -image=fyneio/fyne-cross:android-latest-fix cmd/stoandroid/
//
// then reduce .apk size
// $ zip fyne-cross/dist/android/varasto.apk -d 'lib/armeabi-v7a/*' 'lib/x86/*' 'lib/x86_64/*'
//
// then install:
// $ adb install -r fyne-cross/dist/android/varasto.apk

func main() {
	if err := logic(context.Background()); err != nil {
		panic(err)
	}
}

func logic(ctx context.Context) error {
	app := app.New()

	win := app.NewWindow("Varasto")

	makeTapMenu := func(label string, tap func()) *fyne.Menu {
		menu := fyne.NewMenu(label)
		menu.Tap = tap // hacked patch to allow top-level items (TODO: add actual menuitems top-level)
		return menu
	}

	winCtx := &winContext{ctx, app, win}

	win.SetMainMenu(&fyne.MainMenu{
		Items: []*fyne.Menu{
			makeTapMenu("Home", func() { toHomePage(winCtx) }),
			makeTapMenu("Settings", func() { settingsPage(winCtx) }),
		},
	})

	toHomePage(winCtx)

	// blocks indefinitely
	win.ShowAndRun()

	return nil
}

func toHomePage(winCtx *winContext) {
	/*
		duplicateScannerTarget := func(loc duplicateScannerLocation) *widget.Button {
			return widget.NewButton(loc.Title(), func() {
				duplicateScanner(winCtx, loc)
			})
		}
	*/

	logo := canvas.NewImageFromResource(resourceLogoInkscapeSvg)
	// logo.SetMinSize(fyne.NewSize(100, 100))
	logo.SetMinSize(fyne.NewSize(50, 50))
	// logo.SetMinSize(fyne.NewSize(0, 50)) // squished vertically
	// logo.SetMinSize(fyne.NewSize(0,winCtx.win.Canvas().Size().Height)) // haven't tried yet
	// logo.SetMinSize(fyne.NewSize(winCtx.win.Canvas().Size().Width, 0)) // zero size
	// logo.SetMinSize(fyne.NewSize(100, 0)) // doesn't work
	// logo.SetMinSize(fyne.NewSize(0, 100)) // works, but too big
	// logo.FillMode = canvas.ImageFillContain

	varastoUrl, _ := url.Parse("https://function61.com/varasto")

	version := widget.NewLabel("Ver. dev")
	version.Alignment = fyne.TextAlignTrailing

	winCtx.win.SetContent(vertically(
		logo,
		widget.NewButton("Browse local files", func() {
			browsePage(winCtx)
		}),
		/*
			widget.NewButton("Upload VID_20190310_134312.mp4", func() {
				uploadPage(winCtx, "/storage/emulated/0/DCIM/Camera/VID_20190310_134312.mp4")
			}),
				widget.NewButton("Upload 2", func() {
					uploadPage(winCtx, "/storage/emulated/0/DCIM/Camera/VID_20190310_140702.mp4")
				}),
			duplicateScannerTarget(camera()),
			duplicateScannerTarget(camScanner()),
		*/
		widget.NewButton("Settings", func() {
			settingsPage(winCtx)
		}),
		fyne.NewContainerWithLayout(
			layout.NewGridLayoutWithColumns(2),
			widget.NewHyperlink("function61.com/varasto", varastoUrl),
			version),
	))
}

// encapsulates Fyne window and cancellation context
type winContext struct {
	ctx context.Context
	app fyne.App
	win fyne.Window
}

func createHeading(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

/*
	conf, err := stoclient.ReadConfig()
*/

/*
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
*/

// returns "cache"
// homeDir := a.Storage().RootURI().Name()

// tg:=widget.NewTextGrid()

/*
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			displayError(err)
			return
		}
	}, w)
*/

/*
	pathUnescaped, err := url.PathUnescape(uri.Name())
	if err != nil {
		panic(err)
	}

	// URI is relative to this
	// https://imnotyourson.com/which-storage-directory-should-i-use-for-storing-on-android-6/
	dir := filepath.Join("/storage/emulated/0", pathUnescaped)

	hello.SetText(dir)
*/
