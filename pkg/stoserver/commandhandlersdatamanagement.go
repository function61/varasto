package stoserver

import (
	"errors"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) VolumeMarkDataLost(cmd *stoservertypes.VolumeMarkDataLost, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		volToPurge, err := stodb.Read(tx).Volume(cmd.Id)
		if err != nil {
			return err
		}

		volToPurge.BlobSizeTotal = 0
		volToPurge.BlobCount = 0

		if err := stodb.VolumeRepository.Update(volToPurge, tx); err != nil {
			return err
		}

		return stodb.BlobRepository.Each(func(record interface{}) error {
			blob := record.(*stotypes.Blob)

			writtenAndPendingVolumes := func() int {
				return len(blob.Volumes) + len(blob.VolumesPendingReplication)
			}

			volumesBefore := writtenAndPendingVolumes()

			blob.Volumes = sliceutil.FilterInt(blob.Volumes, func(volId int) bool {
				return volId != volToPurge.ID
			})

			blob.VolumesPendingReplication = sliceutil.FilterInt(blob.VolumesPendingReplication, func(volId int) bool {
				return volId != volToPurge.ID
			})

			// optimization to not save unchanged
			if volumesBefore == writtenAndPendingVolumes() { // volume purge did not affect this?
				return nil
			}

			if cmd.OnlyIfRedundancy && len(blob.Volumes) == 0 {
				return errors.New("aborting because blob would lose last redundant copy")
			}

			return stodb.BlobRepository.Update(blob, tx)
		}, tx)
	})
}
