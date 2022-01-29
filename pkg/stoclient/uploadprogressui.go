package stoclient

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/function61/varasto/pkg/tui"
	"github.com/mattn/go-isatty"
	"github.com/nsf/termbox-go"
	"github.com/olekukonko/tablewriter"
)

type FileUploadProgress struct {
	filePath            string
	bytesInFileTotal    int64
	bytesUploadedInBlob int64 // 0 when we get report of file upload starting
	started             time.Time
	completed           time.Time
}

func (f FileUploadProgress) BytesUploadedInBlob() int64 {
	return f.bytesUploadedInBlob
}

type UploadProgressListener interface {
	// can be called concurrently
	ReportUploadProgress(FileUploadProgress)
	// it is not safe to call ReportUploadProgress after calling Close.
	// returns only after resources (like termbox) used by listener are freed.
	Close()
}

type uploadProgressTextUi struct {
	progress chan FileUploadProgress
	stop     chan interface{}
	stopped  chan interface{}
}

func newUploadProgressTextUi() *uploadProgressTextUi {
	p := &uploadProgressTextUi{
		progress: make(chan FileUploadProgress),
		stop:     make(chan interface{}),
		stopped:  make(chan interface{}),
	}

	go func() {
		if err := p.run(); err != nil {
			panic(err)
		}
	}()

	return p
}

func (p *uploadProgressTextUi) ReportUploadProgress(e FileUploadProgress) {
	p.progress <- e
}

func (p *uploadProgressTextUi) Close() {
	close(p.stop)

	<-p.stopped
}

// runs in separate goroutine
func (p *uploadProgressTextUi) run() error {
	defer func() { close(p.stopped) }()

	// while using termbox, ctrl+c doesn't work as a SIGINT anymore:
	//   https://github.com/nsf/termbox-go/issues/50#issuecomment-60668910
	err := termbox.Init()
	if err != nil {
		return err
	}
	defer termbox.Close()

	inprogressFilesFromEvents := NewFileCollectionUploadStatus()

	drawProgress := func(files []*ObjectUploadStatus) error {
		renderedTbl := &bytes.Buffer{}

		tblBuilder := tablewriter.NewWriter(renderedTbl)
		tblBuilder.SetAutoFormatHeaders(false)
		tblBuilder.SetBorder(false)
		tblBuilder.SetHeader([]string{"File", "Progress", "Speed"})

		for _, file := range files {
			tblBuilder.Append([]string{
				file.Key,
				tui.ProgressBar(int(100.0*float64(file.BytesUploadedTotal)/float64(file.BytesInFileTotal)), 20, tui.ProgressBarCirclesTheme()),
				file.SpeedMbps(),
			})
		}

		tblBuilder.Render()

		if err := termbox.Clear(termbox.ColorDefault, termbox.ColorDefault); err != nil {
			return err
		}

		p.drawLinesToTerminal(strings.Split(renderedTbl.String(), "\n"))

		return termbox.Flush()
	}

	// first draw of UI
	if err := drawProgress(nil); err != nil {
		return err
	}

	for {
		select {
		case <-p.stop:
			return nil
		case progress := <-p.progress:
			if err := inprogressFilesFromEvents.Observe(progress, drawProgress); err != nil {
				return err
			}
		}
	}
}

func (p *uploadProgressTextUi) drawLinesToTerminal(lines []string) {
	for j, line := range lines {
		lineAsRunes := []rune(line)

		for i := 0; i < len(lineAsRunes); i++ {
			termbox.SetCell(i, j, lineAsRunes[i], termbox.ColorDefault, termbox.ColorDefault)
		}
	}
}

type nullUploadProgressListener string

func (n *nullUploadProgressListener) ReportUploadProgress(FileUploadProgress) {}
func (n *nullUploadProgressListener) Close()                                  {}

func NewNullUploadProgressListener() UploadProgressListener {
	x := nullUploadProgressListener("")
	return &x
}

func textUiUploadProgressOutputIfInTerminal() UploadProgressListener {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return newUploadProgressTextUi()
	} else {
		return NewNullUploadProgressListener()
	}
}

type uploadProgressCustomUI struct {
	progress               chan FileUploadProgress
	collectionUploadStatus *FileCollectionUploadStatus
}

func NewUploadProgressCustomUI(onChange func([]*ObjectUploadStatus) error) UploadProgressListener {
	p := &uploadProgressCustomUI{
		progress:               make(chan FileUploadProgress),
		collectionUploadStatus: NewFileCollectionUploadStatus(),
	}

	go func() {
		for e := range p.progress {
			_ = p.collectionUploadStatus.Observe(e, onChange)
		}
	}()

	return p
}

// uploaders call this concurrently
func (p *uploadProgressCustomUI) ReportUploadProgress(e FileUploadProgress) {
	p.progress <- e
}

func (p *uploadProgressCustomUI) Close() {
	close(p.progress)
}
