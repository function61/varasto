// logic for importing/exporting the metadata database into a file
package stodbimportexport

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"go.etcd.io/bbolt"
	"io"
	"regexp"
	"strings"
)

func Export(tx *bbolt.Tx, output io.Writer) error {
	outputBuffered := bufio.NewWriterSize(output, 1024*100)
	defer outputBuffered.Flush()

	nodeId, err := stodb.CfgNodeId.GetRequired(tx)
	if err != nil {
		return err
	}

	backupHeader := backupHeaderJson{NodeId: nodeId, SchemaVersion: stodb.CurrentSchemaVersion}

	if _, err := outputBuffered.Write([]byte(makeBackupHeader(backupHeader) + "\n")); err != nil {
		return err
	}

	jsonEncoderOutput := json.NewEncoder(outputBuffered)

	for heading, repo := range stodb.RepoByRecordType {
		// print heading
		if _, err := outputBuffered.Write([]byte("\n# " + heading + "\n")); err != nil {
			return err
		}

		if err := repo.Each(func(record interface{}) error {
			if err := jsonEncoderOutput.Encode(record); err != nil {
				return err
			}

			return nil
		}, tx); err != nil {
			return err
		}
	}

	return nil
}

func Import(content io.Reader, dbLocation string) error {
	exists, err := fileexists.Exists(dbLocation)
	if exists || err != nil {
		return fmt.Errorf(
			"bailing out for safety because database already exists in %s\npro-tip: rename previous DB to %s.backup to start with a fresh import-able database",
			dbLocation,
			dbLocation)
	}

	db, err := stodb.Open(dbLocation)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := func() error {
		tx, err := db.Begin(true)
		if err != nil {
			return err
		}
		defer func() { ignoreError(tx.Rollback()) }()

		if err := stodb.BootstrapRepos(tx); err != nil {
			return err
		}

		return tx.Commit()
	}(); err != nil {
		return err
	}

	var openTx *bbolt.Tx

	commitOpenTx := func() error {
		if openTx == nil {
			return nil
		}

		return openTx.Commit()
	}

	txUseCount := 0

	// automatically commits every N calls
	withTx := func(fn func(tx *bbolt.Tx) error) error {
		txUseCount++

		if (txUseCount % 2000) == 0 {
			if err := commitOpenTx(); err != nil {
				return err
			}

			openTx = nil

			fmt.Printf(".")
		}

		if openTx == nil {
			var errTxOpen error
			openTx, errTxOpen = db.Begin(true)
			if errTxOpen != nil {
				return errTxOpen
			}
		}

		return fn(openTx)
	}

	defer func() {
		if openTx == nil {
			return
		}

		ignoreError(openTx.Rollback())
	}()

	if err := importDbInternal(content, withTx); err != nil {
		return err
	}

	return commitOpenTx()
}

func importDbInternal(content io.Reader, withTx func(fn func(tx *bbolt.Tx) error) error) error {
	scanner := bufio.NewScanner(content)

	// by default craps out on lines > 64k. set max line to many megabytes
	buf := make([]byte, 0, 8*1024*1024)
	scanner.Buffer(buf, cap(buf))

	var repo blorm.Repository

	// get first line so we can parse the header
	if !scanner.Scan() {
		return fmt.Errorf("file seems empty: %v", scanner.Err())
	}
	snapshotMetadata, err := parseBackupHeader(scanner.Text())
	if err != nil {
		return err
	}

	if snapshotMetadata.SchemaVersion != stodb.CurrentSchemaVersion {
		return fmt.Errorf(
			"snapshot schema version %d while DB requires %d",
			snapshotMetadata.SchemaVersion,
			stodb.CurrentSchemaVersion)
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			recordType := line[2:]

			var found bool
			repo, found = stodb.RepoByRecordType[recordType]
			if !found {
				return fmt.Errorf("unsupported record type: %s", recordType)
			}
		} else {
			// init empty record
			record := repo.Alloc()

			if err := json.Unmarshal([]byte(line), record); err != nil {
				return err
			}

			if err := withTx(func(tx *bbolt.Tx) error {
				return repo.Update(record, tx)
			}); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func makeBackupHeader(details backupHeaderJson) string {
	detailsJson, err := json.Marshal(&details)
	if err != nil {
		panic(err)
	}

	return "# Varasto-DB-snapshot" + string(detailsJson)
}

type backupHeaderJson struct {
	NodeId        string `json:"node_id"`
	SchemaVersion int    `json:"schema_version"`
}

var backupHeaderRe = regexp.MustCompile(`# Varasto-DB-snapshot(\{.+)`)

// returns nodeId
func parseBackupHeader(backupHeader string) (*backupHeaderJson, error) {
	matches := backupHeaderRe.FindStringSubmatch(backupHeader)
	if matches == nil {
		return nil, errors.New("failed to recognize backup header. did you remember to decrypt the backup file?")
	}

	details := &backupHeaderJson{}
	if err := json.Unmarshal([]byte(matches[1]), details); err != nil {
		return nil, err
	}

	return details, nil
}

func ignoreError(err error) {
	// no-op
}
