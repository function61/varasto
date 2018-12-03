package bupserver

import (
	"encoding/json"
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"io"
)

// I have confidence on the robustness of the blobdriver interface, but not yet on the
// robustness of the metadata database. that's why we have this export endpoint - to get
// backups. more confidence will come when this whole system is hooked up to Event Horizon

func exportDb(tx storm.Node, out io.Writer) error {
	type exporter struct {
		name   string
		target interface{}
	}

	exporters := []exporter{
		{"Node", &buptypes.Node{}},
		{"ReplicationPolicy", &buptypes.ReplicationPolicy{}},
		{"Volume", &buptypes.Volume{}},
		{"Collection", &buptypes.Collection{}},
		{"Blob", &buptypes.Blob{}},
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
