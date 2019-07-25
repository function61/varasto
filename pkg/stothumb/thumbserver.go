package stothumb

import (
	"encoding/hex"
	"fmt"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/stopper"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

// you can ask as fast as you want for making thumbs for entire collection, but only one task
// will be launched and will get a channel you can listen to for waiting it to be ready
type ThumbParallelProcessor struct {
	mu                    sync.Mutex
	inProgressCollections map[string]chan interface{}
}

func NewThumbParallelProcessor() *ThumbParallelProcessor {
	return &ThumbParallelProcessor{
		inProgressCollections: map[string]chan interface{}{},
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
		if err := makeThumbsForCollection(collectionId); err != nil {
			log.Printf("FAIL makeThumbsForCollection: %v", err)
		}

		close(done)

		x.collectionDownloadDone(collectionId)
	}()

	return done
}

func runServer(stop *stopper.Stopper) error {
	srv := http.Server{
		Addr: ":8688",
	}

	thumbProcessor := NewThumbParallelProcessor()

	http.HandleFunc("/thumb", func(w http.ResponseWriter, r *http.Request) {
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

		io.Copy(w, thumbFile)
	})

	go func() {
		defer stop.Done()

		<-stop.Signal

		if err := srv.Shutdown(nil); err != nil {
			panic(err)
			// logl.Error.Fatalf("Shutdown() failed: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
