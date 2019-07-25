package stoserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/asdine/storm/codec"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"go.etcd.io/bbolt"
	"io"
	"strings"
)

// I have confidence on the robustness of the blobdriver interface, but not yet on the
// robustness of the metadata database. that's why we have this export endpoint - to get
// backups. more confidence will come when this whole system is hooked up to Event Horizon.
// Run this with:
// 	$ curl -H "Authorization: Bearer $BUP_AUTHTOKEN" http://localhost:8066/api/db/export

func exportDb(tx *bolt.Tx, output io.Writer) error {
	outputBuffered := bufio.NewWriterSize(output, 1024*100)
	defer outputBuffered.Flush()

	jsonEncoderOutput := json.NewEncoder(outputBuffered)

	for heading, repo := range repoByRecordType {
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

func importDb(content io.Reader, nodeId string) error {
	scf, err := readServerConfigFile()
	if err != nil {
		return err
	}

	exists, err := fileexists.Exists(scf.DbLocation)
	if exists || err != nil {
		return fmt.Errorf("bailing out for safety because database already exists in %s", scf.DbLocation)
	}

	db, err := boltOpen(scf)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := bootstrapRepos(db); err != nil {
		return err
	}

	var openTx *bolt.Tx

	commitOpenTx := func() error {
		if openTx == nil {
			return nil
		}

		return openTx.Commit()
	}

	txUseCount := 0

	// automatically commits every N calls
	withTx := func(fn func(tx *bolt.Tx) error) error {
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

		openTx.Rollback()
	}()

	if err := importDbInternal(content, withTx); err != nil {
		return err
	}

	if err := withTx(func(tx *bolt.Tx) error {
		return bootstrapSetNodeId(nodeId, tx)
	}); err != nil {
		return err
	}

	return commitOpenTx()
}

var msgpackCodec codec.MarshalUnmarshaler = msgpack.Codec

// key is heading in export file under which all JSON records are dumped
var repoByRecordType = map[string]blorm.Repository{
	"Blob":                     stodb.BlobRepository,
	"Client":                   stodb.ClientRepository,
	"Collection":               stodb.CollectionRepository,
	"Directory":                stodb.DirectoryRepository,
	"IntegrityVerificationJob": stodb.IntegrityVerificationJobRepository,
	"Node":                     stodb.NodeRepository,
	"ReplicationPolicy":        stodb.ReplicationPolicyRepository,
	"Volume":                   stodb.VolumeRepository,
	"VolumeMount":              stodb.VolumeMountRepository,
}

func importDbInternal(content io.Reader, withTx func(fn func(tx *bolt.Tx) error) error) error {
	scanner := bufio.NewScanner(content)

	// by default craps out on lines > 64k. set max line to many megabytes
	buf := make([]byte, 0, 8*1024*1024)
	scanner.Buffer(buf, cap(buf))

	var repo blorm.Repository

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			recordType := line[2:]

			var found bool
			repo, found = repoByRecordType[recordType]
			if !found {
				return fmt.Errorf("unsupported record type: %s", recordType)
			}
		} else {
			// init empty record
			record := repo.Alloc()

			if err := json.Unmarshal([]byte(line), record); err != nil {
				return err
			}

			if err := withTx(func(tx *bolt.Tx) error {
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
