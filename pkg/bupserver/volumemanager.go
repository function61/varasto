package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
)

func volumeManagerIncreaseBlobCount(tx storm.Node, volumeId string, blobSizeBytes int64) error {
	var volume buptypes.Volume
	if err := tx.One("ID", volumeId, &volume); err != nil {
		return err
	}

	volume.BlobCount++
	volume.BlobSizeTotal += blobSizeBytes

	return tx.Save(&volume)
}
