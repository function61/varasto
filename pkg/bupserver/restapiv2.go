package bupserver

import (
	"bytes"
	"encoding/base64"
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/stateresolver"
	"github.com/function61/eventkit/event"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type handlers struct {
	db *storm.DB
}

func convertDir(dir buptypes.Directory) Directory {
	return Directory{
		Id:     dir.ID,
		Parent: dir.Parent,
		Name:   dir.Name,
	}
}

func convertDbCollection(coll buptypes.Collection, changesets []ChangesetSubset) CollectionSubset {
	return CollectionSubset{
		Id:                coll.ID,
		Directory:         coll.Directory,
		Name:              coll.Name,
		ReplicationPolicy: coll.ReplicationPolicy,
		Changesets:        changesets,
	}
}

func (h *handlers) GetDirectory(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *DirectoryOutput {
	dirId := mux.Vars(r)["id"]

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dir, err := QueryWithTx(tx).Directory(dirId)
	panicIfError(err)

	parentDirs, err := getParentDirs(*dir, tx)
	panicIfError(err)

	parentDirsConverted := []Directory{}

	for _, parentDir := range parentDirs {
		parentDirsConverted = append(parentDirsConverted, convertDir(parentDir))
	}

	dbColls := []buptypes.Collection{}
	if err := tx.Find("Directory", dir.ID, &dbColls); err != nil && err != storm.ErrNotFound {
		panic(err)
	}

	colls := []CollectionSubset{}
	for _, dbColl := range dbColls {
		colls = append(colls, convertDbCollection(dbColl, nil)) // FIXME: nil ok?
	}

	dbSubDirs := []buptypes.Directory{}
	if err := tx.Find("Parent", dir.ID, &dbSubDirs); err != nil && err != storm.ErrNotFound {
		panic(err)
	}

	subDirs := []Directory{}
	for _, dbSubDir := range dbSubDirs {
		subDirs = append(subDirs, convertDir(dbSubDir))
	}

	return &DirectoryOutput{
		Directory:   convertDir(*dir),
		Parents:     parentDirsConverted,
		Directories: subDirs,
		Collections: colls,
	}
}

func (h *handlers) GetCollectiotAtRev(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *CollectionOutput {
	collectionId := mux.Vars(r)["id"]
	changesetId := mux.Vars(r)["rev"]
	pathBytes, err := base64.StdEncoding.DecodeString(mux.Vars(r)["path"])
	if err != nil {
		panic(err)
	}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	coll, err := QueryWithTx(tx).Collection(collectionId)
	if err != nil {
		if err == ErrDbRecordNotFound {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return nil
	}

	if changesetId == "head" {
		changesetId = coll.Head
	}

	state, err := stateresolver.ComputeStateAt(*coll, changesetId)
	panicIfError(err)

	allFilesInRevision := state.FileList()

	// peek brings a subset of allFilesInRevision
	peekResult := stateresolver.DirPeek(allFilesInRevision, string(pathBytes))

	totalSize := int64(0)
	convertedFiles := []File{}

	for _, file := range allFilesInRevision {
		totalSize += file.Size
	}

	for _, file := range peekResult.Files {
		convertedFiles = append(convertedFiles, File{
			Path:     file.Path,
			Sha256:   file.Sha256,
			Created:  file.Created,
			Modified: file.Modified,
			Size:     int(file.Size), // FIXME
			BlobRefs: file.BlobRefs,
		})
	}

	changesetsConverted := []ChangesetSubset{}

	for _, changeset := range coll.Changesets {
		changesetsConverted = append(changesetsConverted, ChangesetSubset{
			Id:      changeset.ID,
			Parent:  changeset.Parent,
			Created: changeset.Created,
		})
	}

	return &CollectionOutput{
		TotalSize: int(totalSize), // FIXME
		SelectedPathContents: SelectedPathContents{
			Path:       peekResult.Path,
			Files:      convertedFiles,
			ParentDirs: peekResult.ParentDirs,
			SubDirs:    peekResult.SubDirs,
		},
		FileCount:   len(allFilesInRevision),
		ChangesetId: changesetId,
		Collection:  convertDbCollection(*coll, changesetsConverted),
	}
}

func (h *handlers) GetVolumes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]Volume {
	ret := []Volume{}

	dbObjects := []buptypes.Volume{}
	panicIfError(h.db.All(&dbObjects))

	for _, dbObject := range dbObjects {
		ret = append(ret, Volume{
			Id:            dbObject.ID,
			Uuid:          dbObject.UUID,
			Label:         dbObject.Label,
			Quota:         int(dbObject.Quota), // FIXME: lossy conversions here
			BlobSizeTotal: int(dbObject.BlobSizeTotal),
			BlobCount:     int(dbObject.BlobCount),
		})
	}

	return &ret
}

func (h *handlers) GetVolumeMounts(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]VolumeMount {
	ret := []VolumeMount{}

	dbObjects := []buptypes.VolumeMount{}
	panicIfError(h.db.All(&dbObjects))

	for _, dbObject := range dbObjects {
		ret = append(ret, VolumeMount{
			Id:         dbObject.ID,
			Volume:     dbObject.Volume,
			Node:       dbObject.Node,
			Driver:     string(dbObject.Driver), // FIXME: string enum to frontend
			DriverOpts: dbObject.DriverOpts,
		})
	}

	return &ret
}

func (h *handlers) GetReplicationPolicies(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]ReplicationPolicy {
	ret := []ReplicationPolicy{}

	dbObjects := []buptypes.ReplicationPolicy{}
	panicIfError(h.db.All(&dbObjects))

	for _, dbObject := range dbObjects {
		ret = append(ret, ReplicationPolicy{
			Id:             dbObject.ID,
			Name:           dbObject.Name,
			DesiredVolumes: dbObject.DesiredVolumes,
		})
	}

	return &ret
}

func (h *handlers) GetNodes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]Node {
	ret := []Node{}

	dbObjects := []buptypes.Node{}
	panicIfError(h.db.All(&dbObjects))

	for _, dbObject := range dbObjects {
		ret = append(ret, Node{
			Id:   dbObject.ID,
			Addr: dbObject.Addr,
			Name: dbObject.Name,
		})
	}

	return &ret
}

func (h *handlers) GetClients(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]Client {
	ret := []Client{}

	dbObjects := []buptypes.Client{}
	panicIfError(h.db.All(&dbObjects))

	for _, dbObject := range dbObjects {
		ret = append(ret, Client{
			Id:        dbObject.ID,
			Name:      dbObject.Name,
			AuthToken: dbObject.AuthToken,
		})
	}

	return &ret
}

// func createNonPersistingEventLog(listeners domain.EventListener) (eventlog.Log, error) {
func createNonPersistingEventLog() (eventlog.Log, error) {
	return eventlog.NewSimpleLogFile(
		bytes.NewReader(nil),
		ioutil.Discard,
		func(e event.Event) error {
			return nil
			// return domain.DispatchEvent(e, listeners)
		},
		func(serialized string) (event.Event, error) {
			return nil, nil
			// return event.Deserialize(serialized, domain.Allocators)
		},
		logex.Discard)
}

func createDummyMiddlewares() httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
	}
}

func getParentDirs(of buptypes.Directory, tx storm.Node) ([]buptypes.Directory, error) {
	parentDirs := []buptypes.Directory{}

	current := &of
	var err error

	for current.Parent != "" {
		current, err = QueryWithTx(tx).Directory(current.Parent)
		if err != nil {
			return nil, err
		}

		// reverse order
		parentDirs = append([]buptypes.Directory{*current}, parentDirs...)
	}

	return parentDirs, nil
}
