package varastothumb

// below side effects have to be imported to transparently support their decoding

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/disintegration/imageorient"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/varastoclient"
	"github.com/function61/varasto/pkg/varastotypes"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

func makeThumbsForCollection(collectionId string) error {
	cc, err := varastoclient.ReadConfig()
	if err != nil {
		return err
	}

	coll, err := varastoclient.FetchCollectionMetadata(*cc, collectionId)
	if err != nil {
		return err
	}

	state, err := stateresolver.ComputeStateAt(*coll, coll.Head)
	if err != nil {
		return err
	}

	errors := uint64(0)

	work := make(chan varastotypes.File, 2)
	workersDone := sync.WaitGroup{}
	worker := func() {
		defer workersDone.Done()

		for file := range work {
			if err := makeThumbForFile(file, *cc); err != nil {
				log.Printf("makeThumbForFile: %s: %v", file.Path, err)
				atomic.AddUint64(&errors, 1)
			}
		}
	}

	for i := 0; i < cap(work); i++ {
		workersDone.Add(1)
		go worker()
	}

	for _, file := range state.FileList() {
		ext := strings.ToLower(filepath.Ext(file.Path))

		makeThumbnail := false

		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
			makeThumbnail = true
		}

		// TODO: file size limit

		if !makeThumbnail {
			continue
		}

		work <- file
	}

	close(work)
	workersDone.Wait()

	if errors > 0 {
		return fmt.Errorf("encountered %d error(s)", errors)
	}

	return nil
}

func genThumbPath(fileContentSha256 []byte) string {
	asBase64 := base64.RawURLEncoding.EncodeToString(fileContentSha256)

	return filepath.Join(
		"thumbs",
		asBase64[0:2],
		fmt.Sprintf("%s.jpg", asBase64[2:len(asBase64)]))
}

// possible outcomes:
// - thumb written succesfully
// - error making thumb - same should not be tried again for this file
// - thumb already exists
func makeThumbForFile(file varastotypes.File, config varastoclient.ClientConfig) error {
	fileContentSha256, err := hex.DecodeString(file.Sha256)
	if err != nil {
		return err
	}

	thumbPath := genThumbPath(fileContentSha256)

	if exists, err := fileexists.Exists(thumbPath); err != nil || exists {
		if err != nil { // error with file exists check
			return err
		}

		return nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
		return err
	}

	log.Printf("Thumbnailing %s", file.Path)

	origBuffer := &bytes.Buffer{}
	if err := varastoclient.DownloadOneFile(file, origBuffer, config); err != nil {
		return err
	}

	// needed to correctly open JPEGs with EXIF "you should rotate this image" -metadata
	orig, _, err := imageorient.Decode(origBuffer)
	if err != nil {
		// let's leave a 0-length thumbnail to indicate that the source file
		// could not be thumbnailed
		if errTruncate := os.Truncate(thumbPath, 0); errTruncate != nil {
			return fmt.Errorf("truncate: %v; tried that due to %v", errTruncate, err)
		}

		return err
	}
	origBuffer = nil // release memory early

	origBounds := orig.Bounds()

	// - NearestNeighbor is fast but usually looks worst.
	// - CatmullRom is slow but usually looks best.
	// - ApproxBiLinear has reasonable speed and quality.
	thumbWidth, thumbHeight := resizedDimensions(origBounds.Max.X, origBounds.Max.Y, 300, 533)

	thumb := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))

	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), orig, origBounds, draw.Over, nil)

	// TODO: we should use a temp-file scheme
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer thumbFile.Close()

	return jpeg.Encode(thumbFile, thumb, nil)
}

func resizedDimensions(width, height, targetw, targeth int) (int, int) {
	return resizedDimensionsInternal(
		float64(width),
		float64(height),
		float64(targetw),
		float64(targeth))
}

func resizedDimensionsInternal(width, height, targetw, targeth float64) (int, int) {
	ratiow := targetw / width
	ratioh := targeth / height

	return int(width * math.Min(ratiow, ratioh)), int(height * math.Min(ratiow, ratioh))
}
