package stoserver

import (
	"path/filepath"
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func TestFrom6To7IntegrityVerificationIssues(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const blobRef = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	legacyReport := "blob " + blobRef + ": read failed\n" +
		"blob " + blobRef + " size mismatch; expected=10 got=9\n" +
		"Completed with 2 error(s)\n" +
		"maximum errors detected; aborting job\n"

	if err := db.Update(func(tx *bbolt.Tx) error {
		if err := stodb.IntegrityVerificationJobRepository.Bootstrap(tx); err != nil {
			return err
		}

		return stodb.IntegrityVerificationJobRepository.Update(&stotypes.IntegrityVerificationJob{
			ID:          "job",
			Deprecated1: legacyReport,
		}, tx)
	}); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(from6to7); err != nil {
		t.Fatal(err)
	}

	if err := db.View(func(tx *bbolt.Tx) error {
		job := &stotypes.IntegrityVerificationJob{}
		if err := stodb.IntegrityVerificationJobRepository.OpenByPrimaryKey([]byte("job"), job, tx); err != nil {
			return err
		}

		assert.Assert(t, len(job.Issues) == 3)
		assert.EqualString(t, job.Issues[0].Blob.AsHex(), blobRef)
		assert.EqualString(t, job.Issues[0].Problem, "read failed")
		assert.EqualString(t, job.Issues[1].Blob.AsHex(), blobRef)
		assert.EqualString(t, job.Issues[1].Problem, "size mismatch; expected=10 got=9")
		assert.Assert(t, job.Issues[2].Blob == nil)
		assert.EqualString(t, job.Issues[2].Problem, "maximum errors detected; aborting job")
		assert.EqualString(t, job.Deprecated1, "")

		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
