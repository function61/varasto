package stoclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/retry"
	"github.com/function61/varasto/pkg/mutexmap"
	"github.com/function61/varasto/pkg/stotypes"
)

type BlobDiscoveredListener interface {
	BlobDiscovered(blobDiscoveredAttrs)
	// listener (like backgroundUploader) will inform its producer (blob discoverer) that
	// uploads are erroring, to request that blob discovery should be stopped
	CancelCh() chan any
}

func NewBlobDiscoveredAttrs(
	ref stotypes.BlobRef,
	collectionID string,
	content []byte,
	maybeCompressible bool,
	filePath string,
	size int64,
) blobDiscoveredAttrs {
	return blobDiscoveredAttrs{
		ref:               ref,
		collectionID:      collectionID,
		content:           content,
		maybeCompressible: maybeCompressible,
		filePath:          filePath,
		size:              size,
	}
}

type blobDiscoveredAttrs struct {
	ref               stotypes.BlobRef
	collectionID      string
	content           []byte
	maybeCompressible bool
	filePath          string
	size              int64
}

type blobDiscoveredNoopListener struct{}

// new blob? didn't do nuffin
func NewBlobDiscoveredNoopListener() BlobDiscoveredListener {
	return &blobDiscoveredNoopListener{}
}

func (b *blobDiscoveredNoopListener) BlobDiscovered(_ blobDiscoveredAttrs) {}

func (b *blobDiscoveredNoopListener) CancelCh() chan any {
	return nil
}

type backgroundUploader struct {
	ctx                  context.Context
	uploadJobs           chan blobDiscoveredAttrs
	clientConfig         ClientConfig
	uploadersDone        chan error
	cancelCh             chan any
	uploadProgress       UploadProgressListener
	blobAlreadyUploading *mutexmap.M // keyed by blob ref
}

func NewBackgroundUploader(
	ctx context.Context,
	n int,
	clientConfig ClientConfig,
	uploadProgress UploadProgressListener,
) *backgroundUploader {
	b := &backgroundUploader{
		ctx:                  ctx,
		uploadJobs:           make(chan blobDiscoveredAttrs),
		uploadersDone:        make(chan error, n),
		clientConfig:         clientConfig,
		cancelCh:             make(chan any),
		uploadProgress:       uploadProgress,
		blobAlreadyUploading: mutexmap.New(),
	}

	for i := 0; i < n; i++ {
		go func() {
			b.uploadersDone <- b.runOneUploader()
		}()
	}

	return b
}

// might block while uploader slots become available
// errors are reported later by WaitFinished()
func (b *backgroundUploader) BlobDiscovered(attrs blobDiscoveredAttrs) {
	// send "0 bytes uploaded" progress event so UI starts showing 0 % for this file,
	// because the next event is sent after blob is uploaded
	b.uploadProgress.ReportUploadProgress(fileProgressEvent{
		filePath:            attrs.filePath,
		bytesInFileTotal:    attrs.size,
		bytesUploadedInBlob: 0,
	})

	b.uploadJobs <- attrs
}

func (b *backgroundUploader) CancelCh() chan any {
	return b.cancelCh
}

// returns error if any of the uploaders encountered error
func (b *backgroundUploader) WaitFinished() error {
	n := cap(b.uploadersDone)

	close(b.uploadJobs)

	for i := 0; i < n; i++ {
		if err := <-b.uploadersDone; err != nil {
			return fmt.Errorf("at least one uploader encountered job error: %v", err)
		}
	}

	b.uploadProgress.Close()

	return nil
}

func (b *backgroundUploader) runOneUploader() error {
	var err error

	for job := range b.uploadJobs {
		// on first error just start dropping jobs on the floor
		if err != nil {
			continue
		}

		err = b.upload(job)
		if err != nil {
			log.Printf("backgroundUploader: %v", err)

			// FIXME: this is not threadsafe
			select {
			case <-b.cancelCh: // already closed
			default: // not closed
				close(b.cancelCh)
			}
		}
	}

	return err
}

func (b *backgroundUploader) upload(job blobDiscoveredAttrs) error {
	unlock, ok := b.blobAlreadyUploading.TryLock(job.ref.AsHex())
	if !ok {
		log.Printf("another thread is already uploading %s", job.ref.AsHex())
		// we'll consider uploading this duplicate blob as a success, even though we don't
		// know if the other thread will succeed doing it, but reporting that error is
		// the responsibility of the other thread
		return nil
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(b.ctx, 3*60*time.Second)
	defer cancel()

	return retry.Retry(ctx, func(ctx context.Context) error {
		return b.uploadInternal(ctx, job)
	}, retry.DefaultBackoff(), func(err error) {
		log.Printf("try failure: %v", err)
	})
}

func (b *backgroundUploader) uploadInternal(ctx context.Context, job blobDiscoveredAttrs) error {
	started := time.Now()

	// just check if the chunk exists already
	blobAlreadyExists, err := blobExists(job.ref, b.clientConfig)
	if err != nil {
		return err
	}

	notifyProgress := func() {
		b.uploadProgress.ReportUploadProgress(fileProgressEvent{
			filePath:            job.filePath,
			bytesInFileTotal:    job.size,
			bytesUploadedInBlob: int64(len(job.content)),
			started:             started,
			completed:           time.Now(),
		})
	}

	if blobAlreadyExists {
		log.Printf("Deduplicated chunk %s", job.ref.AsHex())

		notifyProgress()

		return nil
	}

	// 10 seconds can be too fast waiting for HDD to spin up + blob write
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	if err := UploadBlob(
		ctx,
		job.ref,
		bytes.NewBuffer(job.content),
		job.collectionID,
		job.maybeCompressible,
		b.clientConfig,
	); err != nil {
		return fmt.Errorf("blob %s: %v", job.ref.AsHex(), err)
	}

	notifyProgress()

	return nil
}

func UploadBlob(
	ctx context.Context,
	blobRef stotypes.BlobRef,
	content io.Reader,
	collectionID string,
	maybeCompressible bool,
	clientConfig ClientConfig,
) error {
	// 10 seconds can be too fast waiting for HDD to spin up + blob write
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	if _, err := ezhttp.Post(
		ctx,
		clientConfig.URLBuilder().UploadBlob(blobRef.AsHex(), collectionID, boolToStr(maybeCompressible)),
		ezhttp.AuthBearer(clientConfig.AuthToken),
		ezhttp.SendBody(content, "application/octet-stream"),
		ezhttp.Client(clientConfig.HTTPClient()),
	); err != nil {
		return fmt.Errorf("UploadBlob %s: %v", blobRef.AsHex(), err)
	}

	return nil
}
