package stoclient

// Data structure for tracking file upload statuses of the upload operation as a whole

import (
	"fmt"
	"time"
)

type ObjectUploadStatus struct {
	Key                string // usually file path, but doesn't necessarily need to exist in local filesystem
	BytesInFileTotal   int64
	BytesUploadedTotal int64
	speedMeasurements  []FileUploadProgress
}

func (f *ObjectUploadStatus) Progress() float64 {
	return float64(f.BytesUploadedTotal) / float64(f.BytesInFileTotal)
}

func (f *ObjectUploadStatus) SpeedMbps() string {
	if len(f.speedMeasurements) == 0 {
		return "0 Mbps"
	}

	minTs := f.speedMeasurements[0].started
	maxTs := f.speedMeasurements[0].completed

	totalBytes := int64(0)

	for _, measurement := range f.speedMeasurements {
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

// computes from stream of FileUploadProgress events which files have started uploading, tracks their
// progress and completion
type FileCollectionUploadStatus struct {
	files []*ObjectUploadStatus
}

func NewFileCollectionUploadStatus() *FileCollectionUploadStatus {
	return &FileCollectionUploadStatus{
		files: []*ObjectUploadStatus{},
	}
}

// observes a new progress event, and fires onChange if was interesting enough to warrant updating the UI.
// onChange may only access statuses before returning
func (f *FileCollectionUploadStatus) Observe(
	progress FileUploadProgress,
	onChange func([]*ObjectUploadStatus) error,
) error {
	if f.observe(progress, time.Now()) {
		return onChange(f.files)
	} else {
		return nil
	}
}

// for testing
func (f *FileCollectionUploadStatus) observe(
	progress FileUploadProgress,
	now time.Time,
) bool {
	status, idx, isPreviouslyUnseenFile := func() (*ObjectUploadStatus, int, bool) {
		for idx, fus := range f.files {
			if fus.Key == progress.filePath {
				return fus, idx, false
			}
		}

		f.files = append(f.files, &ObjectUploadStatus{ // have not seen this file before
			Key:               progress.filePath,
			BytesInFileTotal:  progress.bytesInFileTotal,
			speedMeasurements: []FileUploadProgress{},
		})

		idx := len(f.files) - 1

		return f.files[idx], idx, true
	}()

	status.BytesUploadedTotal += progress.bytesUploadedInBlob

	completed := status.BytesUploadedTotal >= status.BytesInFileTotal

	if completed { // delete
		f.files = append(f.files[:idx], f.files[idx+1:]...)
	} else if progress.bytesUploadedInBlob != 0 { // 0 when we get report of file upload starting
		measurements := []FileUploadProgress{progress}

		since := func(t time.Time) time.Duration { return now.Sub(t) }

		// keep only previous measurements for the last N seconds
		for _, previousMeasurement := range status.speedMeasurements {
			if since(previousMeasurement.completed) > 5*time.Second {
				continue // delete it
			}

			measurements = append(measurements, previousMeasurement)
		}

		status.speedMeasurements = measurements
	}

	// for a file with 100 blobs, we get 100 events with bytesUploadedInBlob=0. we are only interested
	// in the first "blob starting" event
	actualChanges := isPreviouslyUnseenFile || progress.bytesUploadedInBlob != 0

	return actualChanges
}
