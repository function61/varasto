package stomediascanner

// below side effects have to be imported to transparently support their decoding

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imageorient"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

func collectionThumbnails(
	ctx context.Context,
	collectionId string,
	moveNamedThumbnails bool,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) error {
	for {
		more, err := collectionThumbnailsOneBatch(
			ctx,
			collectionId,
			moveNamedThumbnails,
			conf,
			logl,
		)
		if err != nil {
			return err
		}

		if !more {
			return nil
		}
	}
}

// bool return is "more" - whether we need to call this worker again to continue with
// more batches
func collectionThumbnailsOneBatch(
	ctx context.Context,
	collectionId string,
	moveNamedThumbnails bool,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) (bool, error) {
	more := false

	blobUploader := stoclient.NewBackgroundUploader(
		ctx,
		stoclient.BackgroundUploaderConcurrency,
		*conf,
		stoclient.NewNullUploadProgressListener())

	coll, err := stoclient.FetchCollectionMetadata(*conf, collectionId)
	if err != nil {
		return more, err
	}

	collHead, err := stateresolver.ComputeStateAtHead(*coll)
	if err != nil {
		return more, err
	}

	collFiles := collHead.Files()

	createdFiles := []stotypes.File{}
	deletedFiles := []string{}

	_, hasBanner := collFiles[stoservertypes.BannerPath]

	if !hasBanner {
		bannerUrl, err := discoverBannerUrl(ctx, coll, conf)
		if err != nil {
			// not worth stopping mediascanner for
			logl.Error.Printf("discoverBannerUrl: %v", err)
		}

		if bannerUrl != "" {
			bannerImage, err := downloadImage(ctx, bannerUrl)
			if err != nil {
				logl.Error.Printf("downloadImage: %v", err)
			} else {
				defer bannerImage.Close()

				fileModifiedTime := time.Now()

				bannerFile, err := stoclient.ScanAndDiscoverBlobs(
					ctx,
					stoservertypes.BannerPath,
					bannerImage,
					0,
					fileModifiedTime, // created
					fileModifiedTime, // modified
					collectionId,
					blobUploader,
				)
				if err != nil {
					return more, err
				}

				createdFiles = append(createdFiles, *bannerFile)
			}
		}
	}

	if moveNamedThumbnails {
		for _, file := range collFiles {
			// do not touch meta files (could already be thumbnails etc)
			if strings.HasPrefix(file.Path, ".sto/") {
				continue
			}

			// only process non-thumbnailable files like videos
			if thumbnailable(file.Path) {
				continue
			}

			// if this is a file that has a corresponding <filename>.jpg
			// (like "funny video.mp4" has "funny video.jpg") assume this file is its thumbnail,
			// and if it is
			ourExt := filepath.Ext(file.Path)

			// "funny video.mp4" => "funny video"
			usWithoutExt := file.Path[0 : len(file.Path)-len(ourExt)]

			// "funny video.mp4" => "funny video.jpg"
			thumbnailCounterpart := usWithoutExt + ".jpg"

			logl.Debug.Printf("thumbnailCounterpart<%s>", thumbnailCounterpart)

			if counterpart, has := collFiles[thumbnailCounterpart]; has {
				// TODO: support moves? right now we're doing delete + create
				// TODO: if we auto-made thumbnail for (now-)thumbnail, remove the thumbnail
				//       for thumbnail?

				// delete counterpart from original location
				deletedFiles = append(deletedFiles, counterpart.Path)

				// move into thumb path, assume it has sensible dimensions (suitable as a thumbnail)
				counterpart.Path = collectionThumbPath(file)

				createdFiles = append(createdFiles, counterpart)

				// delete from file map so the thumbnailer range won't come across our thumbnail
				// again.
				// deletion while iterating is safe: https://stackoverflow.com/a/23230406
				delete(collFiles, file.Path)
			}
		}
	}

	alreadyThumbnailed := func(file stotypes.File) bool {
		thumbPath := collectionThumbPath(file)

		// since thumbPath is based on file content, "foo.jpg" and "foo (Copy).jpg" have
		// same thumb path (if they have same content)
		if _, alreadyCommitted := collFiles[thumbPath]; alreadyCommitted {
			return true
		}

		// these are files that we created just now for this soon-to-commit.
		for _, createdFile := range createdFiles {
			if createdFile.Path == thumbPath {
				return true
			}
		}

		return false
	}

	for _, file := range collFiles {
		// do not touch meta files (could already be thumbnails etc)
		if strings.HasPrefix(file.Path, ".sto/") {
			continue
		}

		if !thumbnailable(file.Path) {
			continue
		}

		if alreadyThumbnailed(file) {
			continue
		}

		maxBatchSize := 100

		if len(createdFiles) > maxBatchSize {
			more = true
			logl.Info.Printf("max batch size of %d reached - doing intermediate commit", maxBatchSize)
			break
		}

		logl.Debug.Printf("thumbnailing %s\n", file.Path)

		fileModifiedTime := time.Now()

		thumbOutput := &bytes.Buffer{}

		if err := makeThumbForFile(ctx, file, thumbOutput, collectionId, *conf); err != nil {
			logl.Error.Printf("makeThumbForFile %s: %v", file.Path, err)
			continue
		}

		createdThumbnail, err := stoclient.ScanAndDiscoverBlobs(
			ctx,
			collectionThumbPath(file),
			thumbOutput,
			0,
			fileModifiedTime, // created
			fileModifiedTime, // modified
			collectionId,
			blobUploader,
		)
		if err != nil {
			// TODO: skip-and-log or not? (like we do above)
			return more, err
		}

		createdFiles = append(createdFiles, *createdThumbnail)
	}

	if err := blobUploader.WaitFinished(); err != nil {
		return more, err
	}

	if len(createdFiles) == 0 {
		return more, nil // no-op
	}

	changeset := stotypes.NewChangeset(
		stoutils.NewCollectionChangesetId(),
		coll.Head,
		time.Now(),
		createdFiles,
		[]stotypes.File{},
		deletedFiles)

	_, err = stoclient.Commit(
		changeset,
		collectionId,
		*conf)

	return more, err
}

func collectionThumbPath(f stotypes.File) string {
	return ".sto/thumb/" + f.Sha256[0:10] + ".jpg"
}

// possible outcomes:
// - thumb written succesfully
// - error making thumb - same should not be tried again for this file
// - thumb already exists
func makeThumbForFile(
	ctx context.Context,
	file stotypes.File,
	thumbOutput io.Writer,
	collectionId string,
	clientConfig stoclient.ClientConfig,
) error {
	origBuffer := &bytes.Buffer{}
	if err := stoclient.DownloadOneFile(ctx, file, collectionId, origBuffer, clientConfig); err != nil {
		return err
	}

	// needed to correctly open JPEGs with EXIF "you should rotate this image" -metadata
	orig, _, err := imageorient.Decode(origBuffer)
	if err != nil {
		return err // TODO: how to react?
	}

	origBounds := orig.Bounds()

	thumbWidth, thumbHeight := resizedDimensions(origBounds.Max.X, origBounds.Max.Y, 300, 533)

	thumb := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))

	// - NearestNeighbor is fast but usually looks worst.
	// - CatmullRom is slow but usually looks best.
	// - ApproxBiLinear has reasonable speed and quality.
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), orig, origBounds, draw.Over, nil)

	return jpeg.Encode(thumbOutput, thumb, nil)
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

func thumbnailable(filePath string) bool {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func downloadImage(ctx context.Context, imageUrl string) (io.ReadCloser, error) {
	resp, err := ezhttp.Get(ctx, imageUrl)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", imageUrl, err)
	}

	typ := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(typ, "image/") {
		return nil, fmt.Errorf("response type not image/*; got %s", typ)
	}

	return resp.Body, nil
}
