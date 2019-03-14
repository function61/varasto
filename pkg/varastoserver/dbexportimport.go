package varastoserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/asdine/storm"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/varasto/pkg/varastotypes"
	"io"
	"strings"
)

// I have confidence on the robustness of the blobdriver interface, but not yet on the
// robustness of the metadata database. that's why we have this export endpoint - to get
// backups. more confidence will come when this whole system is hooked up to Event Horizon.
// Run this with:
// 	$ curl -H "Authorization: Bearer $BUP_AUTHTOKEN" http://localhost:8066/api/db/export

func exportDb(tx storm.Node, out io.Writer) error {
	type exporter struct {
		name   string
		target interface{}
	}

	exporters := []exporter{
		{"Node", &varastotypes.Node{}},
		{"Client", &varastotypes.Client{}},
		{"ReplicationPolicy", &varastotypes.ReplicationPolicy{}},
		{"Volume", &varastotypes.Volume{}},
		{"VolumeMount", &varastotypes.VolumeMount{}},
		{"Directory", &varastotypes.Directory{}},
		{"Collection", &varastotypes.Collection{}},
		{"Blob", &varastotypes.Blob{}},
	}

	enc := json.NewEncoder(out)
	for _, exporter := range exporters {
		out.Write([]byte("\n# " + exporter.name + "\n"))

		if err := exportTable(tx, exporter.target, enc, out); err != nil {
			return err
		}
	}

	return nil
}

func exportTable(tx storm.Node, target interface{}, enc *json.Encoder, out io.Writer) error {
	return tx.Select().Each(target, func(record interface{}) error {
		return enc.Encode(record)
	})
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

	db, err := stormOpen(scf)
	if err != nil {
		return err
	}
	defer db.Close()

	var openTx storm.Node

	commitOpenTx := func() error {
		if openTx == nil {
			return nil
		}

		return openTx.Commit()
	}

	txUseCount := 0

	// automatically commits every N calls
	withTx := func(fn func(tx storm.Node) error) error {
		txUseCount++

		if (txUseCount % 1000) == 0 {
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

		if err := openTx.Rollback(); err != nil {
			panic(fmt.Errorf("rollback failed: %v", err))
		}
	}()

	if err := importDbInternal(content, withTx); err != nil {
		return err
	}

	if err := withTx(func(tx storm.Node) error {
		return tx.Set("settings", "nodeId", nodeId)
	}); err != nil {
		return err
	}

	return commitOpenTx()
}

func importDbInternal(content io.Reader, withTx func(fn func(tx storm.Node) error) error) error {
	scanner := bufio.NewScanner(content)

	// by default craps out on lines > 64k. set max line to many megabytes
	buf := make([]byte, 0, 8*1024*1024)
	scanner.Buffer(buf, cap(buf))

	allocators := map[string]func() interface{}{
		"Node":              func() interface{} { return &varastotypes.Node{} },
		"Client":            func() interface{} { return &varastotypes.Client{} },
		"ReplicationPolicy": func() interface{} { return &varastotypes.ReplicationPolicy{} },
		"Volume":            func() interface{} { return &varastotypes.Volume{} },
		"VolumeMount":       func() interface{} { return &varastotypes.VolumeMount{} },
		"Directory":         func() interface{} { return &varastotypes.Directory{} },
		"Collection":        func() interface{} { return &varastotypes.Collection{} },
		"Blob":              func() interface{} { return &varastotypes.Blob{} },
	}

	typeOfNextLine := ""

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			typeOfNextLine = line[2:]
		} else {
			allocator, found := allocators[typeOfNextLine]
			if !found {
				return fmt.Errorf("allocator not found for: %s", typeOfNextLine)
			}

			// init empty record
			record := allocator()
			if err := json.Unmarshal([]byte(line), record); err != nil {
				return err
			}

			if err := withTx(func(tx storm.Node) error {
				return tx.Save(record)
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
