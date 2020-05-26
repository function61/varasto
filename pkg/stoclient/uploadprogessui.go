package stoclient

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/function61/varasto/pkg/tui"
	"github.com/mattn/go-isatty"
	"github.com/nsf/termbox-go"
	"github.com/olekukonko/tablewriter"
)

type fileProgressEvent struct {
	filePath            string
	bytesInFileTotal    int64
	bytesUploadedInBlob int64 // 0 when we get report of file upload starting
	started             time.Time
	completed           time.Time
}

type UploadProgressListener interface {
	ReportUploadProgress(fileProgressEvent)
	// it is not safe to call ReportUploadProgress after calling Close.
	// returns only after resources (like termbox) used by listener are freed.
	Close()
}

type uploadProgressTextUi struct {
	progress chan fileProgressEvent
	stop     chan interface{}
	stopped  chan interface{}
}

func newUploadProgressTextUi() *uploadProgressTextUi {
	p := &uploadProgressTextUi{
		progress: make(chan fileProgressEvent),
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

func (p *uploadProgressTextUi) ReportUploadProgress(e fileProgressEvent) {
	p.progress <- e
}

func (p *uploadProgressTextUi) Close() {
	close(p.stop)

	<-p.stopped
}

type fileUploadStatus struct {
	filePath           string
	bytesInFileTotal   int64
	bytesUploadedTotal int64
	speedMeasurements  []fileProgressEvent
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

	files := map[string]*fileUploadStatus{}

	drawProgress := func() error {
		renderedTbl := &bytes.Buffer{}

		tblBuilder := tablewriter.NewWriter(renderedTbl)
		tblBuilder.SetAutoFormatHeaders(false)
		tblBuilder.SetBorder(false)
		tblBuilder.SetHeader([]string{"File", "Progress", "Speed"})

		for _, file := range files {
			tblBuilder.Append([]string{
				file.filePath,
				tui.ProgressBar(int(100.0*float64(file.bytesUploadedTotal)/float64(file.bytesInFileTotal)), 20, tui.ProgressBarCirclesTheme()),
				speedMbps(file.speedMeasurements),
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
	if err := drawProgress(); err != nil {
		return err
	}

	for {
		select {
		case <-p.stop:
			return nil
		case progress := <-p.progress:
			status, statusFound := files[progress.filePath]
			if !statusFound { // new file to keep track of
				status = &fileUploadStatus{
					filePath:          progress.filePath,
					bytesInFileTotal:  progress.bytesInFileTotal,
					speedMeasurements: []fileProgressEvent{},
				}

				files[progress.filePath] = status
			}

			status.bytesUploadedTotal += progress.bytesUploadedInBlob

			completed := status.bytesUploadedTotal >= status.bytesInFileTotal

			if completed {
				delete(files, progress.filePath)
			} else if progress.bytesUploadedInBlob != 0 { // 0 when we get report of file upload starting
				measurements := []fileProgressEvent{progress}

				// keep only previous measurements for the last N seconds
				for _, previousMeasurement := range status.speedMeasurements {
					if previousMeasurement.completed.Before(time.Now().Add(-5 * time.Second)) {
						continue // delete it
					}

					measurements = append(measurements, previousMeasurement)
				}

				status.speedMeasurements = measurements
			}

			// don't draw for "0 bytes uploaded" event, because for file with 100 blobs,
			// we get 100 events with bytesUploadedInBlob=0. we are only interested in the
			// first one which notifies of the new file upload starting
			if !statusFound || progress.bytesUploadedInBlob != 0 {
				if err := drawProgress(); err != nil {
					return err
				}
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

func speedMbps(measurements []fileProgressEvent) string {
	if len(measurements) == 0 {
		return "0 Mbps"
	}

	minTs := measurements[0].started
	maxTs := measurements[0].completed

	totalBytes := int64(0)

	for _, measurement := range measurements {
		if measurement.started.Before(minTs) {
			minTs = measurement.started
		}
		if measurement.completed.After(maxTs) {
			maxTs = measurement.completed
		}

		totalBytes += measurement.bytesUploadedInBlob
	}

	// duration in which totalBytes was transferred
	duration := maxTs.Sub(minTs)

	return fmt.Sprintf("%.2f Mbps", float64(totalBytes)/1024.0/1024.0*8.0/float64(duration/time.Second))
}

type nullUploadProgressListener string

func (n *nullUploadProgressListener) ReportUploadProgress(fileProgressEvent) {}
func (n *nullUploadProgressListener) Close()                                 {}

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
