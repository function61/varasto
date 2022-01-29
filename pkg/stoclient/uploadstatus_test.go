package stoclient

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/byteshuman"
)

func TestFileCollectionUploadStatus(t *testing.T) {
	inprogressFiles := NewFileCollectionUploadStatus()

	pendingFilesUI := func() string { // simulation for UI
		lines := []string{}
		for _, f := range inprogressFiles.files {
			lines = append(lines, fmt.Sprintf("\n%s %d/%d", f.Key, f.BytesUploadedTotal, f.BytesInFileTotal))
		}
		return strings.Join(lines, "")
	}

	t0 := time.Date(2020, 1, 7, 14, 45, 0, 0, time.UTC)
	tplus1 := t0.Add(1 * time.Second)

	observeUpload := func(filePath string, bytesUploadedInBlob int64) bool {
		return inprogressFiles.observe(FileUploadProgress{
			filePath:            filePath,
			bytesInFileTotal:    8 * byteshuman.MiB,
			bytesUploadedInBlob: bytesUploadedInBlob,
			started:             t0,
			completed:           tplus1,
		}, tplus1)
	}

	assert.Assert(t, observeUpload("foo.txt", 0))
	assert.Assert(t, !observeUpload("foo.txt", 0)) // 2nd "blob starting" => no change
	assert.Assert(t, observeUpload("foo.txt", 4*byteshuman.MiB))

	assert.Assert(t, observeUpload("bar.txt", 0))

	assert.EqualString(t, pendingFilesUI(), `
foo.txt 4194304/8388608
bar.txt 0/8388608`)

	assert.Assert(t, observeUpload("foo.txt", 3*byteshuman.MiB))

	assert.EqualString(t, pendingFilesUI(), `
foo.txt 7340032/8388608
bar.txt 0/8388608`)

	assert.Assert(t, observeUpload("foo.txt", 1*byteshuman.MiB))
	assert.Assert(t, observeUpload("bar.txt", 1*byteshuman.MiB))

	assert.EqualString(t, pendingFilesUI(), `
bar.txt 1048576/8388608`)
}

func TestSpeedMbps(t *testing.T) {
	t0 := time.Date(2019, 11, 4, 15, 32, 0, 0, time.UTC)

	megabytes := func(n int) int64 {
		return int64(1024 * 1024 * n)
	}

	tcs := []struct {
		input          []FileUploadProgress
		expectedOutput string
	}{
		{
			[]FileUploadProgress{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0,
				},
			},
			"+Inf Mbps",
		},
		{
			[]FileUploadProgress{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0.Add(1 * time.Second),
				},
			},
			"32.00 Mbps",
		},
		{
			[]FileUploadProgress{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0.Add(2 * time.Second),
				},
			},
			"16.00 Mbps",
		},
		{
			[]FileUploadProgress{
				{
					bytesUploadedInBlob: megabytes(8),
					started:             t0,
					completed:           t0.Add(1 * time.Second),
				},
				{
					bytesUploadedInBlob: megabytes(8),
					started:             t0,
					completed:           t0.Add(1 * time.Second),
				},
			},
			"128.00 Mbps",
		},
		{
			[]FileUploadProgress{
				{
					bytesUploadedInBlob: megabytes(8),
					started:             t0,
					completed:           t0.Add(1 * time.Second),
				},
				{
					bytesUploadedInBlob: megabytes(8),
					started:             t0,
					completed:           t0.Add(2 * time.Second),
				},
			},
			"64.00 Mbps",
		},
	}

	for _, tc := range tcs {
		tc := tc // pin
		t.Run(tc.expectedOutput, func(t *testing.T) {
			status := ObjectUploadStatus{speedMeasurements: tc.input}
			assert.EqualString(t, status.SpeedMbps(), tc.expectedOutput)
		})
	}
}
