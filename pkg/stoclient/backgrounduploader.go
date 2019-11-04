package stoclient

import (
	"bytes"
	"context"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/retry"
	"github.com/function61/varasto/pkg/mutexmap"
	"github.com/function61/varasto/pkg/stotypes"
	"log"
	"time"
)

type BlobDiscoveredListener interface {
	BlobDiscovered(blobDiscoveredAttrs)
	// listener (like backgroundUploader) will inform its producer (blob discoverer) that
	// uploads are erroring, to request that blob discovery should be stopped
	CancelCh() chan interface{}
}

func NewBlobDiscoveredAttrs(
	ref stotypes.BlobRef,
	collectionId string,
	content []byte,
	maybeCompressible bool,
	filePath string,
	size int64,
) blobDiscoveredAttrs {
	return blobDiscoveredAttrs{
		ref:               ref,
		collectionId:      collectionId,
		content:           content,
		maybeCompressible: maybeCompressible,
		filePath:          filePath,
		size:              size,
	}
}

type blobDiscoveredAttrs struct {
	ref               stotypes.BlobRef
	collectionId      string
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

func (n *blobDiscoveredNoopListener) BlobDiscovered(_ blobDiscoveredAttrs) {}

func (b *blobDiscoveredNoopListener) CancelCh() chan interface{} {
	return nil
}

type backgroundUploader struct {
	uploadJobs           chan blobDiscoveredAttrs
	clientConfig         ClientConfig
	uploadersDone        chan error
	cancelCh             chan interface{}
	uploadProgress       UploadProgressListener
	blobAlreadyUploading *mutexmap.M // keyed by blob ref
}

func NewBackgroundUploader(n int, clientConfig ClientConfig, uploadProgress UploadProgressListener) *backgroundUploader {
	b := &backgroundUploader{
		uploadJobs:           make(chan blobDiscoveredAttrs),
		uploadersDone:        make(chan error, n),
		clientConfig:         clientConfig,
		cancelCh:             make(chan interface{}),
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
		filePath:         attrs.filePath,
		bytesInFileTotal: attrs.size,
	})

	b.uploadJobs <- attrs
}

func (b *backgroundUploader) CancelCh() chan interface{} {
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

	ctx, cancel := context.WithTimeout(context.TODO(), 3*60*time.Second)
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

	if res, err := ezhttp.Post(
		ctx,
		b.clientConfig.UrlBuilder().UploadBlob(job.ref.AsHex(), job.collectionId, boolToStr(job.maybeCompressible)),
		ezhttp.AuthBearer(b.clientConfig.AuthToken),
		ezhttp.SendBody(bytes.NewBuffer(job.content), "application/octet-stream")); err != nil {
		return fmt.Errorf("error uploading blob %s: %v", job.ref.AsHex(), errSample(err, res))
	}

	notifyProgress()

	return nil
}
