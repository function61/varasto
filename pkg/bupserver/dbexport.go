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
		name string
		fn   func(enc *json.Encoder, tx storm.Node, out io.Writer) error
	}

	exporters := []exporter{
		{"Node", exportNodes},
		{"ReplicationPolicy", exportReplicationPolicies},
		{"Volume", exportVolumes},
		{"Collection", exportCollections},
		{"Blob", exportBlobs},
	}

	enc := json.NewEncoder(out)
	for _, exporter := range exporters {
		out.Write([]byte("\n# " + exporter.name + "\n"))

		if err := exporter.fn(enc, tx, out); err != nil {
			return err
		}
	}

	return nil
}

func exportNodes(enc *json.Encoder, tx storm.Node, out io.Writer) error {
	var nodes []*buptypes.Node
	if err := tx.All(&nodes); err != nil {
		return err
	}
	for _, item := range nodes {
		if err := enc.Encode(&item); err != nil {
			return err
		}
	}
	return nil
}

func exportReplicationPolicies(enc *json.Encoder, tx storm.Node, out io.Writer) error {
	var replPolicies []*buptypes.ReplicationPolicy
	if err := tx.All(&replPolicies); err != nil {
		return err
	}
	for _, item := range replPolicies {
		if err := enc.Encode(&item); err != nil {
			return err
		}
	}
	return nil
}

func exportVolumes(enc *json.Encoder, tx storm.Node, out io.Writer) error {
	var volumes []*buptypes.Volume
	if err := tx.All(&volumes); err != nil {
		return err
	}
	for _, item := range volumes {
		if err := enc.Encode(&item); err != nil {
			return err
		}
	}
	return nil
}

func exportCollections(enc *json.Encoder, tx storm.Node, out io.Writer) error {
	var collections []*buptypes.Collection
	if err := tx.All(&collections); err != nil {
		return err
	}
	for _, item := range collections {
		if err := enc.Encode(&item); err != nil {
			return err
		}
	}
	return nil
}

func exportBlobs(enc *json.Encoder, tx storm.Node, out io.Writer) error {
	var blobs []*buptypes.Blob
	if err := tx.All(&blobs); err != nil {
		return err
	}
	for _, item := range blobs {
		if err := enc.Encode(&item); err != nil {
			return err
		}
	}
	return nil
}
