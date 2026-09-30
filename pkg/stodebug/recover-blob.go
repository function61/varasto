package stodebug

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/function61/gokit/atomicfilewrite"
	"github.com/function61/gokit/osutil"
	"github.com/spf13/cobra"
)

func recoverBlobCommand() *cobra.Command {
	recoveredFilename := "recovered.bin"

	cmd := &cobra.Command{
		Use:   "recover-blob [blob-path] [crc32]",
		Short: "Recover a blob with one flipped bit, when knowing its correct CRC32",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			expectedCrc32, err := parseCrc32(args[1])
			osutil.ExitIfError(err)

			osutil.ExitIfError(recoverBlobWithProgress(args[0], expectedCrc32, recoveredFilename, cmd.OutOrStdout()))
		},
	}

	cmd.Flags().StringVarP(&recoveredFilename, "recovered-filename", "", recoveredFilename, "File to write the recovered content to")

	return cmd
}

func recoverBlobWithProgress(path string, expectedCrc32 uint32, recoveredFilename string, progressOutput io.Writer) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if crc32.ChecksumIEEE(content) == expectedCrc32 {
		return fmt.Errorf("blob is already valid")
	}

	totalAttempts := uint64(len(content)) * 8
	if totalAttempts == 0 {
		return errors.New("content looks to be empty")
	}

	// parallelization is important because this takes ~1 h for a 4 MB blob (Ryzen 9 5900X) where corruption is near the end of the stream.
	workerCount := min(runtime.GOMAXPROCS(0), len(content))

	var nextByte atomic.Uint64
	var progressMu sync.Mutex
	completedAttempts := uint64(0)
	lastProgress := uint64(0)
	reportProgress := func() {
		progressMu.Lock()
		defer progressMu.Unlock()

		completedAttempts++
		progress := completedAttempts * 100 / totalAttempts
		for lastProgress < progress {
			lastProgress++
			_, _ = fmt.Fprintf(progressOutput, "\rRecovery progress: %d%%", lastProgress)
		}
	}

	stop := make(chan struct{})
	var foundOnce sync.Once
	found := make(chan []byte, 1)
	finish := func(candidate []byte) {
		foundOnce.Do(func() {
			found <- candidate
			close(stop)
		})
	}

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()

			for {
				byteIndex := int(nextByte.Add(1) - 1)
				if byteIndex >= len(content) {
					return
				}

				candidate := append([]byte(nil), content...)
				for bitIndex := range 8 {
					select {
					case <-stop:
						return
					default:
					}

					candidate[byteIndex] ^= 1 << bitIndex
					matches := crc32.ChecksumIEEE(candidate) == expectedCrc32
					if !matches {
						candidate[byteIndex] ^= 1 << bitIndex
					}
					reportProgress()

					if matches {
						finish(candidate)
						return
					}
				}
			}
		}()
	}

	workers.Wait()
	_, _ = fmt.Fprintln(progressOutput)

	select {
	case recovered := <-found:
		return atomicfilewrite.Write(recoveredFilename, func(w io.Writer) error {
			_, err := io.Copy(w, bytes.NewReader(recovered))
			return err
		})
	default:
		return fmt.Errorf("no single-bit recovery matches CRC32 %08x", expectedCrc32)
	}
}

func parseCrc32(value string) (uint32, error) {
	if len(value) != 8 {
		return 0, fmt.Errorf("CRC32 must be an eight-digit hexadecimal value")
	}

	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse CRC32: %w", err)
	}

	return uint32(parsed), nil
}
