package stoclient

import (
	"github.com/function61/gokit/assert"
	"testing"
	"time"
)

func TestSpeedMbps(t *testing.T) {
	t0 := time.Date(2019, 11, 4, 15, 32, 0, 0, time.UTC)

	megabytes := func(n int) int64 {
		return int64(1024 * 1024 * n)
	}

	tcs := []struct {
		input          []fileProgressEvent
		expectedOutput string
	}{
		{
			[]fileProgressEvent{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0,
				},
			},
			"+Inf Mbps",
		},
		{
			[]fileProgressEvent{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0.Add(1 * time.Second),
				},
			},
			"32.00 Mbps",
		},
		{
			[]fileProgressEvent{
				{
					bytesUploadedInBlob: megabytes(4),
					started:             t0,
					completed:           t0.Add(2 * time.Second),
				},
			},
			"16.00 Mbps",
		},
		{
			[]fileProgressEvent{
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
			[]fileProgressEvent{
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
		t.Run(tc.expectedOutput, func(t *testing.T) {
			assert.EqualString(t, speedMbps(tc.input), tc.expectedOutput)
		})
	}
}
