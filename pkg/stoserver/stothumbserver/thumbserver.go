// Thumbnailer server
package stothumbserver

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoutils"
)

// you can ask as fast as you want for making thumbs for entire collection, but only one task
// will be launched and will get a channel you can listen to for waiting it to be ready
type ThumbParallelProcessor struct {
	mu                    sync.Mutex
	inProgressCollections map[string]chan interface{}
	logl                  *logex.Leveled
	clientConfig          stoclient.ClientConfig
}

func NewThumbParallelProcessor(clientConfig stoclient.ClientConfig, logl *logex.Leveled) *ThumbParallelProcessor {
	return &ThumbParallelProcessor{
		inProgressCollections: map[string]chan interface{}{},
		logl:                  logl,
		clientConfig:          clientConfig,
	}
}

func (x *ThumbParallelProcessor) collectionDownloadDone(collectionId string) {
	x.mu.Lock()
	defer x.mu.Unlock()

	delete(x.inProgressCollections, collectionId)
}

func (x *ThumbParallelProcessor) makeThumbnailsForCollection(collectionId string) chan interface{} {
	x.mu.Lock()
	defer x.mu.Unlock()

	if done, exists := x.inProgressCollections[collectionId]; exists {
		return done
	}

	done := make(chan interface{})

	x.inProgressCollections[collectionId] = done

	go func() {
		if err := makeThumbsForCollection(collectionId, x.clientConfig, x.logl); err != nil {
			x.logl.Error.Printf("makeThumbsForCollection: %v", err)
		}

		close(done)

		x.collectionDownloadDone(collectionId)
	}()

	return done
}

func runServer(addr string, logger *log.Logger, stop *stopper.Stopper) error {
	logl := logex.Levels(logger)

	clientConfig, err := stoclient.ReadConfig()
	if err != nil {
		return err
	}

	thumbProcessor := NewThumbParallelProcessor(*clientConfig, logl)

	http.HandleFunc("/api/thumbnails/thumb", func(w http.ResponseWriter, r *http.Request) {
		collectionId := r.URL.Query().Get("coll")

		if collectionId == "" {
			http.Error(w, "collectionId not specified", http.StatusBadRequest)
			return
		}

		fileSha256, err := hex.DecodeString(r.URL.Query().Get("file"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		thumbPath := genThumbPath(fileSha256)

		if exists, err := fileexists.Exists(thumbPath); err != nil || !exists {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			<-thumbProcessor.makeThumbnailsForCollection(collectionId)
		}

		thumbFile, err := os.Open(thumbPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer thumbFile.Close()

		stat, err := thumbFile.Stat()
		if err != nil {
			panic(err)
		}

		if stat.Size() == 0 { // original file could not be thumbnailed
			http.Error(w, "source file could not be thumbnailed", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

		if _, err := io.Copy(w, thumbFile); err != nil {
			logl.Error.Printf("thumb endpoint write to client: %v", err)
		}
	})

	listener, err := stoutils.CreateTcpOrDomainSocketListener(addr, logl)
	if err != nil {
		return err
	}

	srv := http.Server{}

	go func() {
		defer stop.Done()

		<-stop.Signal

		if err := srv.Shutdown(context.TODO()); err != nil {
			logl.Error.Fatalf("Shutdown() failed: %v", err)
		}
	}()

	logl.Info.Printf("listening on %s", addr)

	if err := srv.Serve(listener); err != http.ErrServerClosed {
		return err
	}

	return nil
}
