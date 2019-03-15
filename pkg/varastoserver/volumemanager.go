package varastoserver

import (
	"go.etcd.io/bbolt"
)

func volumeManagerIncreaseBlobCount(tx *bolt.Tx, volumeId int, blobSizeBytes int64) error {
	volume, err := QueryWithTx(tx).Volume(volumeId)
	if err != nil {
		return err
	}

	volume.BlobCount++
	volume.BlobSizeTotal += blobSizeBytes

	return VolumeRepository.Update(volume, tx)
}

func volumeManagerBestVolumeIdForBlob(candidateVolumes []int, conf *ServerConfig) (int, bool) {
	for _, candidateVolume := range candidateVolumes {
		if _, mountedOnSelfNode := conf.VolumeDrivers[candidateVolume]; mountedOnSelfNode {
			return candidateVolume, true
		}
	}

	return 0, false
}
