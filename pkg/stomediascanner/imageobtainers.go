package stomediascanner

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/function61/gokit/mime"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stomediascanner/pdfthumbnailer"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/nwaples/rardecode"
)

type downloadFileFromVarastoDetails struct {
	file         stotypes.File
	collectionId string
	client       *stoclient.Client
}

type imageObtainer func(context.Context, downloadFileFromVarastoDetails) (io.ReadCloser, error)

// TODO: all obtainers: streaming instead of buffering

// the file is already an image that we can directly open
func alreadyAnImageObtainer(ctx context.Context, varastoFile downloadFileFromVarastoDetails) (io.ReadCloser, error) {
	data := &bytes.Buffer{}

	if err := varastoFile.client.DownloadOneFile(
		ctx,
		varastoFile.collectionId,
		varastoFile.file,
		data,
	); err != nil {
		return nil, err
	}

	return ioutil.NopCloser(data), nil
}

func cbzObtainer(ctx context.Context, varastoFile downloadFileFromVarastoDetails) (io.ReadCloser, error) {
	data := &bytes.Buffer{}

	if err := varastoFile.client.DownloadOneFile(
		ctx,
		varastoFile.collectionId,
		varastoFile.file,
		data,
	); err != nil {
		return nil, err
	}

	cbz, err := zip.NewReader(bytes.NewReader(data.Bytes()), int64(data.Len()))
	if err != nil {
		return nil, err
	}

	if len(cbz.File) == 0 {
		return nil, errors.New("empty zip")
	}

	// TODO: assumption that archive is alphabetically ordered is wrong
	firstFile := cbz.File[0]

	if err := assertFilenameIsImage(firstFile.Name); err != nil {
		return nil, err
	}

	return firstFile.Open()
}

// comic book, RAR variant (RAR archive with images inside)
func cbrObtainer(ctx context.Context, varastoFile downloadFileFromVarastoDetails) (io.ReadCloser, error) {
	data := &bytes.Buffer{}

	if err := varastoFile.client.DownloadOneFile(
		ctx,
		varastoFile.collectionId,
		varastoFile.file,
		data,
	); err != nil {
		return nil, err
	}

	rarPassword := ""
	archive, err := rardecode.NewReader(data, rarPassword)
	if err != nil {
		return nil, fmt.Errorf("rardecode: %w", err)
	}

	// TODO: assumption that archive is alphabetically ordered is wrong
	header, err := archive.Next()
	if err != nil {
		return nil, fmt.Errorf("no first file: %w", err)
	}

	if err := assertFilenameIsImage(header.Name); err != nil {
		return nil, err
	}

	return ioutil.NopCloser(archive), nil
}

func pdfObtainer(ctx context.Context, varastoFile downloadFileFromVarastoDetails) (io.ReadCloser, error) {
	data := &bytes.Buffer{}

	if err := varastoFile.client.DownloadOneFile(
		ctx,
		varastoFile.collectionId,
		varastoFile.file,
		data,
	); err != nil {
		return nil, err
	}

	thumbnailOutput := &bytes.Buffer{}

	if err := pdfthumbnailer.FirstPageAsPng(data, thumbnailOutput); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(thumbnailOutput), nil
	/*
		client, err := pdfrasterizerclient.New(pdfrasterizerclient.Function61, func() (string, error) {
			return "dummytok", nil
		})
		if err != nil {
			return nil, err
		}

		return client.RasterizeToPng(ctx, data)
	*/
}

func assertFilenameIsImage(filename string) error {
	contentType := mime.TypeByExtension(filepath.Ext(filename), mime.NoFallback)

	if !mime.Is(contentType, mime.TypeImage) {
		return fmt.Errorf("%s: not image: %s", filename, contentType)
	}

	return nil
}
