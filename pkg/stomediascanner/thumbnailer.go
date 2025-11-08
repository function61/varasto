package stomediascanner

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
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
	"golang.org/x/image/draw"
)

func collectionThumbnails(
	ctx context.Context,
	collectionID string,
	moveNamedThumbnails bool,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) error {
	for {
		more, err := collectionThumbnailsOneBatch(
			ctx,
			collectionID,
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
	collectionID string,
	moveNamedThumbnails bool,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) (bool, error) {
	more := false

	client := conf.Client()

	blobUploader := stoclient.NewBackgroundUploader(
		ctx,
		stoclient.BackgroundUploaderConcurrency,
		*conf,
		stoclient.NewNullUploadProgressListener())

	coll, err := client.FetchCollectionMetadata(ctx, collectionID)
	if err != nil {
		return more, err
	}

	collHead, err := stateresolver.ComputeStateAtHead(*coll)
	if err != nil {
		return more, err
	}

	collFiles := collHead.Files()

	createdFiles := []stotypes.File{}
	updatedFiles := []stotypes.File{}
	deletedFiles := []string{}

	_, hasBanner := collFiles[stoservertypes.BannerPath]

	if !hasBanner {
		bannerURL, err := discoverBannerURL(ctx, coll, conf, logl)
		if err != nil {
			// not worth stopping mediascanner for
			logl.Error.Printf("discoverBannerUrl: %v", err)
		}

		if bannerURL != "" {
			bannerImage, err := downloadImage(ctx, bannerURL)
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
					collectionID,
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
			if thumbnailable(file.Path) != nil {
				continue
			}

			// if this is a file that has a corresponding <filename>.jpg
			// (like "funny video.mp4" has "funny video.jpg") assume this file is its thumbnail,
			// and if it is
			ourExt := filepath.Ext(file.Path)

			// "funny video.mp4" => "funny video"
			usWithoutExt := file.Path[0 : len(file.Path)-len(ourExt)]

			// "funny video.mp4" => "funny video.jpg"
			thumbnailCounterpartName := usWithoutExt + ".jpg"

			logl.Debug.Printf("thumbnailCounterpart<%s>", thumbnailCounterpartName)

			if thumb, has := collFiles[thumbnailCounterpartName]; has {
				// delete from original location
				deletedFiles = append(deletedFiles, thumb.Path)

				// move into thumb path, assume it has sensible dimensions (suitable as a thumbnail)
				thumbPath := CollectionThumbPath(file)

				// there already exists thumbnail? delete it (by updating)
				if toReplace, thumbExists := collFiles[thumbPath]; thumbExists {
					toReplace.CopyEverythingExceptPath(thumb)

					updatedFiles = append(updatedFiles, toReplace)
				} else {
					thumb.Path = thumbPath
					createdFiles = append(createdFiles, thumb)
				}

				// delete from file map so the thumbnailer range won't come across our thumbnail
				// again.
				// deletion while iterating is safe: https://stackoverflow.com/a/23230406
				delete(collFiles, file.Path)
			}
		}
	}

	alreadyThumbnailed := func(file stotypes.File) bool {
		thumbPath := CollectionThumbPath(file)

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

		imgObtainer := thumbnailable(file.Path)
		if imgObtainer == nil {
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

		if err := makeThumbForFile(ctx, downloadFileFromVarastoDetails{
			file:         file,
			collectionID: collectionID,
			client:       client,
		}, imgObtainer, thumbOutput); err != nil {
			logl.Error.Printf("makeThumbForFile %s: %v", file.Path, err)
			continue
		}

		createdThumbnail, err := stoclient.ScanAndDiscoverBlobs(
			ctx,
			CollectionThumbPath(file),
			thumbOutput,
			0,
			fileModifiedTime, // created
			fileModifiedTime, // modified
			collectionID,
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
		stoutils.NewCollectionChangesetID(),
		coll.Head,
		time.Now(),
		createdFiles,
		updatedFiles,
		deletedFiles)

	_, err = client.Commit(
		ctx,
		collectionID,
		changeset)

	return more, err
}

func CollectionThumbPath(f stotypes.File) string {
	return ".sto/thumb/" + f.Sha256[0:10] + ".jpg"
}

// possible outcomes:
// - thumb written succesfully
// - error making thumb - same should not be tried again for this file
// - thumb already exists
func makeThumbForFile(
	ctx context.Context,
	varastoFile downloadFileFromVarastoDetails,
	imgObtainer imageObtainer,
	thumbOutput io.Writer,
) error {
	imgBytes, err := imgObtainer(ctx, varastoFile)
	if err != nil {
		return err
	}
	defer imgBytes.Close()

	// needed to correctly open JPEGs with EXIF "you should rotate this image" -metadata
	orig, _, err := imageorient.Decode(imgBytes)
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

	return int(width * min(ratiow, ratioh)), int(height * math.Min(ratiow, ratioh))
}

func thumbnailable(filePath string) imageObtainer {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".cbr":
		return cbrObtainer
	case ".cbz":
		return cbzObtainer
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return alreadyAnImageObtainer
	default:
		return nil
	}
}

func downloadImage(ctx context.Context, imageURL string) (io.ReadCloser, error) {
	resp, err := ezhttp.Get(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", imageURL, err)
	}

	typ := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(typ, "image/") {
		return nil, fmt.Errorf("response type not image/*; got %s", typ)
	}

	return resp.Body, nil
}
