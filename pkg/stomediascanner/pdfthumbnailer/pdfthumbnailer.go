package pdfthumbnailer

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/function61/gokit/cryptorandombytes"
)

type GhostScriptDeviceName string

const (
	OutputJpeg GhostScriptDeviceName = "jpeg"   // ghostscript device name
	OutputPng  GhostScriptDeviceName = "png16m" // ghostscript device name
)

func FirstPageAsPng(pdf io.Reader, output io.Writer) error {
	return thumbnailFirstPageInternal(pdf, output, OutputPng)
}

func FirstPageAsJpeg(pdf io.Reader, output io.Writer) error {
	return thumbnailFirstPageInternal(pdf, output, OutputJpeg)
}

// TODO: change API to support streaming
func thumbnailFirstPageInternal(
	pdf io.Reader,
	thumbnailOutput io.Writer,
	gsDevice GhostScriptDeviceName,
) error {
	// needs to be unique for each request. using named pipe because Ghostscript pollutes
	// stdout with log messages, and therefore it doesn't support writing work output there
	outputPath, cleanup, err := randomFifoName()
	if err != nil {
		return err
	}
	defer cleanup()

	// Ghostscript and reading from FIFO need to happen concurrently, b/c writing to
	// the FIFO blocks until the bytes are consumed
	outputDone := runSimpleTaskAsync(func() error {
		ghostscriptOutput, err := os.Open(outputPath)
		if err != nil {
			return err
		}
		defer ghostscriptOutput.Close()

		// send image to client
		_, err = io.Copy(thumbnailOutput, ghostscriptOutput)
		return err
	})

	ghostscript := exec.Command(
		"gs",
		"-dNOPAUSE",
		"-dBATCH",
		"-o", outputPath,
		"-dUseCropBox",
		"-dFirstPage=1", // this and next combined: only read the first page
		"-dLastPage=1",
		"-r300",               // internal rendering DPI
		"-dDownScaleFactor=3", // 300/3 = 100 DPI
		"-sDEVICE="+string(gsDevice),
		"-dJPEGQ=95", // does not error when given to PNG also
		"-",          // = take from stdin
	)
	ghostscript.Stdin = pdf

	if err := ghostscript.Run(); err != nil {
		return fmt.Errorf("ghostscript run: %w", err)
	}

	if err := <-outputDone; err != nil {
		return fmt.Errorf("output: %w", err)
	}

	return nil
}

func randomFifoName() (string, func(), error) {
	randomPath := filepath.Join("/tmp", cryptorandombytes.Base64Url(8))

	if err := syscall.Mkfifo(randomPath, 0600); err != nil {
		return "", nil, err
	}

	return randomPath, func() {
		if err := os.Remove(randomPath); err != nil {
			log.Printf("randomFifoName cleanup: %v", err)
		}
	}, nil
}

func runSimpleTaskAsync(fn func() error) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn()
	}()
	return errCh
}
