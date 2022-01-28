package stoserver

import (
	"strings"

	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoserver/stodb"
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
		Created:           dir.Created,
		Parent:            dir.Parent,
		MetaCollectionId:  dir.MetaCollection,
		Name:              dir.Name,
		ReplicationPolicy: replicationPolicy,
		Type:              typ,
		Sensitivity:       dir.Sensitivity,
	}
}

func convertDbCollection(
	coll stotypes.Collection,
	changesets []stoservertypes.ChangesetSubset,
	state *stateresolver.StateAt,
) *stoservertypes.CollectionSubsetWithMeta {
	encryptionKeyIds := []string{}
	for _, encryptionKey := range coll.EncryptionKeys {
		encryptionKeyIds = append(encryptionKeyIds, encryptionKey.KeyId)
	}

	var rating *int
	if coll.Rating != 0 {
		rating = &coll.Rating
	}

	subset := stoservertypes.CollectionSubset{
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

	filesInMeta := []string{}
	for _, file := range state.FileList() {
		if strings.HasPrefix(file.Path, ".sto/") {
			filesInMeta = append(filesInMeta, file.Path)
		}
	}

	return &stoservertypes.CollectionSubsetWithMeta{
		Collection:    subset,
		FilesInMeta:   filesInMeta,
		FilesInMetaAt: subset.Head,
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

func newDirectoryAndMeta(dir stoservertypes.Directory, tx *bbolt.Tx) (*stoservertypes.DirectoryAndMeta, error) {
	var metaCollection *stoservertypes.CollectionSubsetWithMeta

	if dir.MetaCollectionId != "" {
		metaCollectionDb, err := stodb.Read(tx).Collection(dir.MetaCollectionId)
		if err != nil {
			return nil, err
		}

		state, err := stateresolver.ComputeStateAtHead(*metaCollectionDb)
		if err != nil {
			return nil, err
		}

		metaCollection = convertDbCollection(*metaCollectionDb, []stoservertypes.ChangesetSubset{}, state)
	}

	return &stoservertypes.DirectoryAndMeta{
		Directory:      dir,
		MetaCollection: metaCollection,
	}, nil
}
