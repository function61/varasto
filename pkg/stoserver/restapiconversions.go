package stoserver

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func convertDir(dir stotypes.Directory) stoservertypes.Directory {
	typ, err := stoservertypes.DirectoryTypeValidate(dir.Type)
	if err != nil {
		panic(err)
	}

	var replicationPolicy *string
	if dir.ReplicationPolicy != "" {
		replicationPolicy = &dir.ReplicationPolicy
	}

	return stoservertypes.Directory{
		Id:                dir.ID,
		Parent:            dir.Parent,
		Name:              dir.Name,
		Description:       dir.Description,
		ReplicationPolicy: replicationPolicy,
		Type:              typ,
		Metadata:          metadataMapToKvList(dir.Metadata),
		Sensitivity:       dir.Sensitivity,
	}
}

func convertDbCollection(coll stotypes.Collection, changesets []stoservertypes.ChangesetSubset) stoservertypes.CollectionSubset {
	encryptionKeyIds := []string{}
	for _, encryptionKey := range coll.EncryptionKeys {
		encryptionKeyIds = append(encryptionKeyIds, encryptionKey.KeyId)
	}

	var rating *int
	if coll.Rating != 0 {
		rating = &coll.Rating
	}

	return stoservertypes.CollectionSubset{
		Id:                coll.ID,
		Head:              coll.Head,
		Created:           coll.Created,
		Directory:         coll.Directory,
		Name:              coll.Name,
		Description:       coll.Description,
		ReplicationPolicy: coll.ReplicationPolicy,
		Sensitivity:       coll.Sensitivity,
		EncryptionKeyIds:  encryptionKeyIds,
		Metadata:          metadataMapToKvList(coll.Metadata),
		Tags:              coll.Tags,
		Rating:            rating,
		Changesets:        changesets,
	}
}

func convertFile(file stotypes.File) stoservertypes.File {
	return stoservertypes.File{
		Path:     file.Path,
		Sha256:   file.Sha256,
		Created:  file.Created,
		Modified: file.Modified,
		Size:     int(file.Size), // FIXME
		BlobRefs: file.BlobRefs,
	}
}

func getParentDirsConverted(of stotypes.Directory, tx *bbolt.Tx) ([]stoservertypes.Directory, error) {
	parentDirs, err := getParentDirs(of, tx)
	if err != nil {
		return nil, err
	}

	parentDirsConverted := []stoservertypes.Directory{}

	for _, parentDir := range parentDirs {
		parentDirsConverted = append(parentDirsConverted, convertDir(parentDir))
	}

	return parentDirsConverted, nil
}

func metadataMapToKvList(kvmap map[string]string) []stoservertypes.MetadataKv {
	kvList := []stoservertypes.MetadataKv{}
	for key, value := range kvmap {
		kvList = append(kvList, stoservertypes.MetadataKv{
			Key:   key,
			Value: value,
		})
	}

	return kvList
}
