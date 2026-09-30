package stodebug

import (
	"bytes"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoverBlob(t *testing.T) {
	tempDir := t.TempDir()
	original := []byte("blob content")
	corrupted := append([]byte(nil), original...)
	corrupted[4] ^= 1 << 3
	inputPath := filepath.Join(tempDir, "corrupted.bin")
	if err := os.WriteFile(inputPath, corrupted, 0600); err != nil {
		t.Fatal(err)
	}

	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWorkingDirectory) })

	if err := recoverBlob(inputPath, crc32.ChecksumIEEE(original)); err != nil {
		t.Fatal(err)
	}

	recovered, err := os.ReadFile("recovered.bin")
	if err != nil {
		t.Fatal(err)
	}
	if string(recovered) != string(original) {
		t.Errorf("recovered = %q, want %q", recovered, original)
	}
}

func TestRecoverBlobNoMatch(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "corrupted.bin")
	if err := os.WriteFile(inputPath, []byte("blob content"), 0600); err != nil {
		t.Fatal(err)
	}

	var progress bytes.Buffer
	if err := recoverBlobWithProgress(inputPath, 0, "recovered.bin", &progress); err == nil {
		t.Fatal("recoverBlob() succeeded without a matching checksum")
	}
	if !strings.Contains(progress.String(), "Recovery progress: 1%") {
		t.Errorf("progress did not report 1%%: %q", progress.String())
	}
	if !strings.Contains(progress.String(), "Recovery progress: 100%") {
		t.Errorf("progress did not report 100%%: %q", progress.String())
	}
}

func TestRecoverBlobAlreadyValid(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "valid.bin")
	content := []byte("blob content")
	if err := os.WriteFile(inputPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	err := recoverBlob(inputPath, crc32.ChecksumIEEE(content))
	if err == nil {
		t.Fatal("recoverBlob() succeeded for an already-valid blob")
	}
	if err.Error() != "blob is already valid" {
		t.Errorf("recoverBlob() error = %q, want %q", err, "blob is already valid")
	}
}

func TestParseCrc32(t *testing.T) {
	parsed, err := parseCrc32("c0ffee12")
	if err != nil {
		t.Fatal(err)
	}
	if parsed != 0xc0ffee12 {
		t.Errorf("parseCrc32() = %08x, want c0ffee12", parsed)
	}

	if _, err := parseCrc32("c0ffee1z"); err == nil {
		t.Fatal("parseCrc32() accepted invalid input")
	}
}

func recoverBlob(path string, expectedCrc32 uint32) error {
	return recoverBlobWithProgress(path, expectedCrc32, "recovered.bin", os.Stdout)
}
