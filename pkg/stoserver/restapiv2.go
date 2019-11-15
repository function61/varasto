package stoserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/function61/eventkit/event"
	"github.com/function61/eventkit/eventlog"
	"github.com/function61/eventkit/guts"
	"github.com/function61/gokit/appuptime"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type handlers struct {
	db           *bolt.DB
	conf         *ServerConfig
	ivController *stointegrityverifier.Controller
	logger       *log.Logger
}

func defineRestApi(
	router *mux.Router,
	conf *ServerConfig,
	db *bolt.DB,
	ivController *stointegrityverifier.Controller,
	mwares httpauth.MiddlewareChainMap,
	logger *log.Logger,
) error {
	var han stoservertypes.HttpHandlers = &handlers{db, conf, ivController, logger}

	stoservertypes.RegisterRoutes(han, mwares, muxregistrator.New(router))

	return nil
}

func convertDir(dir stotypes.Directory) stoservertypes.Directory {
	typ, err := stoservertypes.DirectoryTypeValidate(dir.Type)
	if err != nil {
		panic(err)
	}

	return stoservertypes.Directory{
		Id:          dir.ID,
		Parent:      dir.Parent,
		Name:        dir.Name,
		Description: dir.Description,
		Type:        typ,
		Metadata:    metadataMapToKvList(dir.Metadata),
		Sensitivity: dir.Sensitivity,
	}
}

func convertDbCollection(coll stotypes.Collection, changesets []stoservertypes.ChangesetSubset) stoservertypes.CollectionSubset {
	encryptionKeyIds := []string{}
	for _, encryptionKey := range coll.EncryptionKeys {
		encryptionKeyIds = append(encryptionKeyIds, encryptionKey.KeyId)
	}

	return stoservertypes.CollectionSubset{
		Id:               coll.ID,
		Head:             coll.Head,
		Created:          coll.Created,
		Directory:        coll.Directory,
		Name:             coll.Name,
		Description:      coll.Description,
		DesiredVolumes:   coll.DesiredVolumes,
		Sensitivity:      coll.Sensitivity,
		EncryptionKeyIds: encryptionKeyIds,
		Metadata:         metadataMapToKvList(coll.Metadata),
		Tags:             coll.Tags,
		Changesets:       changesets,
	}
}

func (h *handlers) GetDirectory(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.DirectoryOutput {
	dirId := mux.Vars(r)["id"]

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dir, err := stodb.Read(tx).Directory(dirId)
	panicIfError(err)

	parentDirs, err := getParentDirs(*dir, tx)
	panicIfError(err)

	parentDirsConverted := []stoservertypes.Directory{}

	for _, parentDir := range parentDirs {
		parentDirsConverted = append(parentDirsConverted, convertDir(parentDir))
	}

	dbColls, err := stodb.Read(tx).CollectionsByDirectory(dir.ID)
	panicIfError(err)

	colls := []stoservertypes.CollectionSubset{}
	for _, dbColl := range dbColls {
		colls = append(colls, convertDbCollection(dbColl, nil)) // FIXME: nil ok?
	}
	sort.Slice(colls, func(i, j int) bool { return colls[i].Name < colls[j].Name })

	dbSubDirs, err := stodb.Read(tx).SubDirectories(dir.ID)
	panicIfError(err)

	subDirs := []stoservertypes.Directory{}
	for _, dbSubDir := range dbSubDirs {
		subDirs = append(subDirs, convertDir(dbSubDir))
	}
	sort.Slice(subDirs, func(i, j int) bool { return subDirs[i].Name < subDirs[j].Name })

	return &stoservertypes.DirectoryOutput{
		Directory:   convertDir(*dir),
		Parents:     parentDirsConverted,
		Directories: subDirs,
		Collections: colls,
	}
}

func (h *handlers) GetCollectiotAtRev(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.CollectionOutput {
	collectionId := mux.Vars(r)["id"]
	changesetId := mux.Vars(r)["rev"]
	pathBytes, err := base64.StdEncoding.DecodeString(mux.Vars(r)["path"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	coll, err := stodb.Read(tx).Collection(collectionId)
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return nil
	}

	if changesetId == stoservertypes.HeadRevisionId {
		changesetId = coll.Head
	}

	state, err := stateresolver.ComputeStateAt(*coll, changesetId)
	panicIfError(err)

	allFilesInRevision := state.FileList()

	// peek brings a subset of allFilesInRevision
	peekResult := stateresolver.DirPeek(allFilesInRevision, string(pathBytes))

	totalSize := int64(0)
	convertedFiles := []stoservertypes.File{}

	for _, file := range allFilesInRevision {
		totalSize += file.Size
	}

	for _, file := range peekResult.Files {
		convertedFiles = append(convertedFiles, stoservertypes.File{
			Path:     file.Path,
			Sha256:   file.Sha256,
			Created:  file.Created,
			Modified: file.Modified,
			Size:     int(file.Size), // FIXME
			BlobRefs: file.BlobRefs,
		})
	}

	changesetsConverted := []stoservertypes.ChangesetSubset{}

	for _, changeset := range coll.Changesets {
		changesetsConverted = append(changesetsConverted, stoservertypes.ChangesetSubset{
			Id:      changeset.ID,
			Parent:  changeset.Parent,
			Created: changeset.Created,
		})
	}

	return &stoservertypes.CollectionOutput{
		TotalSize: int(totalSize), // FIXME
		SelectedPathContents: stoservertypes.SelectedPathContents{
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

// TODO: URL parameter comes via a hack in frontend
func (h *handlers) DownloadFile(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	collectionId := mux.Vars(r)["id"]
	changesetId := mux.Vars(r)["rev"]

	fileKey := r.URL.Query().Get("file")

	tx, err := h.db.Begin(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	coll, err := stodb.Read(tx).Collection(collectionId)
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, "collection not found", http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	state, err := stateresolver.ComputeStateAt(*coll, changesetId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files := state.Files()
	file, found := files[fileKey]
	if !found {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	type RefAndVolumeId struct {
		Ref      stotypes.BlobRef
		VolumeId int
	}

	refAndVolumeIds := []RefAndVolumeId{}
	for _, refSerialized := range file.BlobRefs {
		ref, err := stotypes.BlobRefFromHex(refSerialized)
		if err != nil { // should not happen
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		blob, err := stodb.Read(tx).Blob(*ref)
		if err != nil {
			if err == blorm.ErrNotFound {
				http.Error(w, "blob not found: "+ref.AsHex(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, "blob pointed to by file metadata not found", http.StatusInternalServerError)
				return
			}
		}

		volumeId, err := h.conf.DiskAccess.BestVolumeId(blob.Volumes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		refAndVolumeIds = append(refAndVolumeIds, RefAndVolumeId{
			Ref:      *ref,
			VolumeId: volumeId,
		})
	}

	tx.Rollback() // eagerly b/c the below operation is slow

	w.Header().Set("Content-Type", contentTypeForFilename(fileKey))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileKey))

	sendBlob := func(refAndVolumeId RefAndVolumeId) error {
		chunkStream, err := h.conf.DiskAccess.Fetch(refAndVolumeId.Ref, coll.EncryptionKeys, refAndVolumeId.VolumeId)
		if err != nil {
			return err
		}
		defer chunkStream.Close()

		_, err = io.Copy(w, chunkStream)
		return err
	}

	for _, refAndVolumeId := range refAndVolumeIds {
		panicIfError(sendBlob(refAndVolumeId))
	}
}

func (h *handlers) GetSubsystemStatuses(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.SubsystemStatus {
	convert := func(subsys *subsystem) stoservertypes.SubsystemStatus {
		status := subsys.controller.Status()

		var started *time.Time
		if !status.Started.IsZero() {
			started = &status.Started
		}

		return stoservertypes.SubsystemStatus{
			Id:          subsys.id,
			Description: status.Description,
			Pid:         status.Pid,
			Alive:       status.Alive,
			HttpMount:   subsys.httpMount,
			Enabled:     subsys.enabled,
			Started:     started,
		}
	}

	return &[]stoservertypes.SubsystemStatus{
		convert(h.conf.ThumbServer),
		convert(h.conf.FuseProjector),
	}
}

func (h *handlers) GetVolumes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.Volume {
	ret := []stoservertypes.Volume{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.Volume{}
	panicIfError(stodb.VolumeRepository.Each(stodb.VolumeAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		var topology *stoservertypes.VolumeTopology

		if dbObject.Enclosure != "" {
			topology = &stoservertypes.VolumeTopology{
				Enclosure: dbObject.Enclosure,
				Slot:      dbObject.EnclosureSlot,
			}
		}

		var latestReport *stoservertypes.SmartReport
		if dbObject.SmartReport != "" {
			latestReport = &stoservertypes.SmartReport{}
			panicIfError(json.Unmarshal([]byte(dbObject.SmartReport), latestReport))
		}

		smartAttrs := stoservertypes.VolumeSmartAttrs{
			Id:           dbObject.SmartId,
			LatestReport: latestReport,
		}

		var mfg *guts.Date
		if !dbObject.Manufactured.IsZero() {
			mfg = &guts.Date{Time: dbObject.Manufactured}
		}

		var we *guts.Date
		if !dbObject.WarrantyEnds.IsZero() {
			we = &guts.Date{Time: dbObject.WarrantyEnds}
		}

		ret = append(ret, stoservertypes.Volume{
			Id:            dbObject.ID,
			Uuid:          dbObject.UUID,
			Label:         dbObject.Label,
			Description:   dbObject.Description,
			SerialNumber:  dbObject.SerialNumber,
			Technology:    stoservertypes.VolumeTechnology(dbObject.Technology),
			Manufactured:  mfg,
			WarrantyEnds:  we,
			Topology:      topology,
			Smart:         smartAttrs,
			Quota:         int(dbObject.Quota), // FIXME: lossy conversions here
			BlobSizeTotal: int(dbObject.BlobSizeTotal),
			BlobCount:     int(dbObject.BlobCount),
		})
	}

	return &ret
}

func (h *handlers) GetVolumeMounts(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.VolumeMount {
	ret := []stoservertypes.VolumeMount{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.VolumeMount{}
	panicIfError(stodb.VolumeMountRepository.Each(stodb.VolumeMountAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		ret = append(ret, stoservertypes.VolumeMount{
			Id:         dbObject.ID,
			Online:     h.conf.DiskAccess.IsMounted(dbObject.Volume),
			Volume:     dbObject.Volume,
			Node:       dbObject.Node,
			Driver:     string(dbObject.Driver), // FIXME: string enum to frontend
			DriverOpts: dbObject.DriverOpts,
		})
	}

	return &ret
}

func (h *handlers) GetBlobMetadata(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.BlobMetadata {
	ref, err := stotypes.BlobRefFromHex(mux.Vars(r)["ref"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	tx, err := h.db.Begin(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	blob, err := stodb.Read(tx).Blob(*ref)
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, "blob not found", http.StatusNotFound)
			return nil
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}
	}

	return &stoservertypes.BlobMetadata{
		Ref:                       blob.Ref.AsHex(),
		Size:                      int(blob.Size),
		SizeOnDisk:                int(blob.SizeOnDisk),
		Referenced:                blob.Referenced,
		IsCompressed:              blob.IsCompressed,
		Volumes:                   blob.Volumes,
		VolumesPendingReplication: blob.VolumesPendingReplication,
	}
}

func toDbFiles(files []stoservertypes.File) []stotypes.File {
	ret := []stotypes.File{}

	for _, file := range files {
		ret = append(ret, stotypes.File{
			Path:     file.Path,
			Sha256:   file.Sha256,
			Created:  file.Created,
			Modified: file.Modified,
			Size:     int64(file.Size),
			BlobRefs: file.BlobRefs,
		})
	}

	return ret
}

func (h *handlers) CommitChangeset(rctx *httpauth.RequestContext, changeset stoservertypes.Changeset, w http.ResponseWriter, r *http.Request) {
	coll := commitChangesetInternal(
		w,
		r,
		mux.Vars(r)["id"],
		stotypes.NewChangeset(
			changeset.ID,
			changeset.Parent,
			changeset.Created,
			toDbFiles(changeset.FilesCreated),
			toDbFiles(changeset.FilesUpdated),
			changeset.FilesDeleted),
		h.db)

	// FIXME: add "produces" to here because commitChangesetInternal responds with updated collection
	if coll != nil {
		// logl.Info.Printf("Collection %s changeset %s committed", coll.ID, changeset.ID)
		log.Printf("Collection %s changeset %s committed", coll.ID, changeset.ID)

		outJson(w, coll)
	}
}

func commitChangesetInternal(w http.ResponseWriter, r *http.Request, collectionId string, changeset stotypes.CollectionChangeset, db *bolt.DB) *stotypes.Collection {
	tx, errTxBegin := db.Begin(true)
	if errTxBegin != nil {
		http.Error(w, errTxBegin.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	coll, err := stodb.Read(tx).Collection(collectionId)
	panicIfError(err)

	if collectionHasChangesetId(changeset.ID, coll) {
		http.Error(w, "changeset ID already in collection", http.StatusBadRequest)
		return nil
	}

	if changeset.Parent != stotypes.NoParentId && !collectionHasChangesetId(changeset.Parent, coll) {
		http.Error(w, "parent changeset not found", http.StatusBadRequest)
		return nil
	}

	if changeset.Parent != coll.Head {
		// TODO: force push or rebase support?
		http.Error(w, "commit does not target current head. would result in dangling heads!", http.StatusBadRequest)
		return nil
	}

	createdAndUpdated := append(changeset.FilesCreated, changeset.FilesUpdated...)

	for _, file := range createdAndUpdated {
		for _, refHex := range file.BlobRefs {
			ref, err := stotypes.BlobRefFromHex(refHex)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return nil
			}

			blob, err := stodb.Read(tx).Blob(*ref)
			if err != nil {
				http.Error(w, fmt.Sprintf("blob %s not found", ref.AsHex()), http.StatusBadRequest)
				return nil
			}

			// FIXME: if same changeset mentions same blob many times, we update the old blob
			// metadata many times due to the transaction reads not seeing uncommitted writes
			blob.Referenced = true
			blob.VolumesPendingReplication = missingFromLeftHandSide(
				blob.Volumes,
				coll.DesiredVolumes)

			// FIXME: temporary limitation
			if !stotypes.HasKeyId(blob.EncryptionKeyId, coll.EncryptionKeys) {
				http.Error(w, "deduplicating Blob? EncryptionKeyId not in coll.EncryptionKeys", http.StatusInternalServerError)
				return nil
			}

			panicIfError(stodb.BlobRepository.Update(blob, tx))
		}
	}

	// update head pointer & calc Created timestamp
	if err := appendAndValidateChangeset(changeset, coll); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	panicIfError(stodb.CollectionRepository.Update(coll, tx))
	panicIfError(tx.Commit())

	return coll
}

func (h *handlers) GetReplicationPolicies(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ReplicationPolicy {
	ret := []stoservertypes.ReplicationPolicy{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.ReplicationPolicy{}
	panicIfError(stodb.ReplicationPolicyRepository.Each(stodb.ReplicationPolicyAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		ret = append(ret, stoservertypes.ReplicationPolicy{
			Id:             dbObject.ID,
			Name:           dbObject.Name,
			DesiredVolumes: dbObject.DesiredVolumes,
		})
	}

	return &ret
}

func (h *handlers) GetReplicationStatuses(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ReplicationStatus {
	statuses := []stoservertypes.ReplicationStatus{}
	for volId, controller := range h.conf.ReplicationControllers {
		statuses = append(statuses, stoservertypes.ReplicationStatus{
			VolumeId: volId,
			Progress: controller.Progress(),
		})
	}

	sort.Slice(statuses, func(i, j int) bool { return statuses[i].VolumeId < statuses[j].VolumeId })

	return &statuses
}

func (h *handlers) GetKeyEncryptionKeys(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.KeyEncryptionKey {
	ret := []stoservertypes.KeyEncryptionKey{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.KeyEncryptionKey{}
	panicIfError(stodb.KeyEncryptionKeyRepository.Each(stodb.KeyEncryptionKeyAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		ret = append(ret, stoservertypes.KeyEncryptionKey{
			Id:          dbObject.ID,
			Kind:        dbObject.Kind,
			Bits:        dbObject.Bits,
			Created:     dbObject.Created,
			Label:       dbObject.Label,
			Fingerprint: dbObject.Fingerprint,
			PublicKey:   dbObject.PublicKey,
		})
	}

	return &ret
}

func (h *handlers) GetNodes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.Node {
	ret := []stoservertypes.Node{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.Node{}
	panicIfError(stodb.NodeRepository.Each(stodb.NodeAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		ret = append(ret, stoservertypes.Node{
			Id:   dbObject.ID,
			Addr: dbObject.Addr,
			Name: dbObject.Name,
		})
	}

	return &ret
}

func (h *handlers) GetApiKeys(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ApiKey {
	ret := []stoservertypes.ApiKey{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.Client{}
	panicIfError(stodb.ClientRepository.Each(stodb.ClientAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		ret = append(ret, stoservertypes.ApiKey{
			Id:        dbObject.ID,
			Created:   dbObject.Created,
			Name:      dbObject.Name,
			AuthToken: dbObject.AuthToken,
		})
	}

	return &ret
}

func (h *handlers) DatabaseExportSha256s(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	w.Header().Set("Content-Type", "text/plain")

	processFile := func(file *stotypes.File) {
		fmt.Fprintf(w, "%s %s\n", file.Sha256, file.Path)
	}

	panicIfError(stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		for _, changeset := range coll.Changesets {
			for _, file := range changeset.FilesCreated {
				processFile(&file)
			}

			for _, file := range changeset.FilesUpdated {
				processFile(&file)
			}
		}

		return nil
	}, tx))
}

// I have confidence on the robustness of the blobdriver interface, but not yet on the
// robustness of the metadata database. that's why we have this export endpoint - to get
// backups. more confidence will come when this whole system is hooked up to Event Horizon.
// Run this with:
// 	$ curl -H "Authorization: Bearer $BUP_AUTHTOKEN" http://localhost:8066/api/db/export
func (h *handlers) DatabaseExport(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	panicIfError(stodbimportexport.Export(tx, w))
}

func (h *handlers) GetLogs(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]string {
	lines := h.conf.LogTail.Snapshot()
	return &lines
}

func (h *handlers) UploadFile(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.File {
	collectionId := mux.Vars(r)["id"]
	mtimeUnixMillis, err := strconv.Atoi(r.URL.Query().Get("mtime"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	mtime := time.Unix(int64(mtimeUnixMillis/1000), 0)

	// TODO: reuse the logic found in the client package?
	wholeFileHash := sha256.New()

	var volumeId int
	if err := h.db.View(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(collectionId)
		if err != nil {
			return err
		}

		volumeId = coll.DesiredVolumes[0]

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	file := &stoservertypes.File{
		Path:     r.URL.Query().Get("filename"),
		Sha256:   "",
		Created:  mtime,
		Modified: mtime,
		Size:     0,
		BlobRefs: []string{},
	}

	for {
		chunk, errRead := ioutil.ReadAll(io.LimitReader(r.Body, stotypes.BlobSize))
		if errRead != nil {
			http.Error(w, errRead.Error(), http.StatusBadRequest)
			return nil
		}

		if len(chunk) == 0 {
			// should only happen if file size is exact multiple of blobSize
			break
		}

		file.Size += len(chunk)

		if _, err := wholeFileHash.Write(chunk); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}

		chunkSha256Bytes := sha256.Sum256(chunk)

		blobRef, err := stotypes.BlobRefFromHex(hex.EncodeToString(chunkSha256Bytes[:]))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}

		blobExists, err := doesBlobExist(*blobRef, h.db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}

		if !blobExists {
			if err := h.conf.DiskAccess.WriteBlob(
				volumeId,
				collectionId,
				*blobRef,
				bytes.NewBuffer(chunk),
				stoutils.IsMaybeCompressible(file.Path),
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil
			}
		}

		file.BlobRefs = append(file.BlobRefs, blobRef.AsHex())
	}

	file.Sha256 = fmt.Sprintf("%x", wholeFileHash.Sum(nil))

	return file
}

func (h *handlers) GetIntegrityVerificationJobs(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.IntegrityVerificationJob {
	ret := []stoservertypes.IntegrityVerificationJob{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer tx.Rollback()

	dbObjects := []stotypes.IntegrityVerificationJob{}
	panicIfError(stodb.IntegrityVerificationJobRepository.Each(stodb.IntegrityVerificationJobAppender(&dbObjects), tx))

	sort.Slice(dbObjects, func(i, j int) bool { return !dbObjects[i].Started.Before(dbObjects[j].Started) })

	runningIds := h.ivController.ListRunningJobs()

	for _, dbObject := range dbObjects {
		completed := dbObject.Completed

		completedPtr := &completed
		if completed.IsZero() {
			completedPtr = nil
		}

		ret = append(ret, stoservertypes.IntegrityVerificationJob{
			Id:                   dbObject.ID,
			Running:              sliceutil.ContainsString(runningIds, dbObject.ID),
			Created:              dbObject.Started,
			Completed:            completedPtr,
			VolumeId:             dbObject.VolumeId,
			LastCompletedBlobRef: dbObject.LastCompletedBlobRef.AsHex(),
			BytesScanned:         int(dbObject.BytesScanned),
			ErrorsFound:          dbObject.ErrorsFound,
			Report:               dbObject.Report,
		})
	}

	return &ret
}

func (h *handlers) GetHealth(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.Health {
	healthRoot, err := getHealthCheckerGraph(h.db, h.conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	graph, err := healthRoot.CheckHealth()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return graph
}

func (h *handlers) GetConfig(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.ConfigValue {
	key := mux.Vars(r)["id"]

	tx, err := h.db.Begin(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	var val string
	switch key {
	case stoservertypes.CfgFuseServerBaseUrl:
		val, err = stodb.CfgFuseServerBaseUrl.GetOptional(tx)
	case stoservertypes.CfgTheMovieDbApikey:
		val, err = stodb.CfgTheMovieDbApikey.GetOptional(tx)
	case stoservertypes.CfgNetworkShareBaseUrl:
		val, err = stodb.CfgNetworkShareBaseUrl.GetOptional(tx)
	case stoservertypes.CfgUbackupConfig:
		val, err = stodb.CfgUbackupConfig.GetOptional(tx)
	case stoservertypes.CfgMetadataLastOk:
		val, err = stodb.CfgMetadataLastOk.GetOptional(tx)
	default:
		http.Error(w, fmt.Sprintf("unknown key: %s", key), http.StatusNotFound)
		return nil
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return &stoservertypes.ConfigValue{
		Key:   key,
		Value: val,
	}
}

func (h *handlers) GetServerInfo(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.ServerInfo {
	dbFileInfo, err := os.Stat(h.conf.File.DbLocation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)

	return &stoservertypes.ServerInfo{
		AppVersion:   dynversion.Version,
		StartedAt:    appuptime.Started(),
		DatabaseSize: int(dbFileInfo.Size()),
		CpuCount:     runtime.NumCPU(),
		ProcessId:    fmt.Sprintf("%d", os.Getpid()),
		HeapBytes:    int(ms.HeapAlloc),
		GoVersion:    runtime.Version(),
		Goroutines:   runtime.NumGoroutine(),
		ServerOs:     runtime.GOOS,
		ServerArch:   runtime.GOARCH,
	}
}

func (h *handlers) GetUbackupStoredBackups(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.UbackupStoredBackup {
	conf, err := ubConfigFromDb(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	backups, err := listUbackupStoredBackups(*conf, h.logger)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return &backups
}

// returns 404 if blob not found
func (h *handlers) DownloadBlob(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	collId := r.URL.Query().Get("collId")

	var coll *stotypes.Collection

	getBlobMetadata := func(blobRefSerialized string) (*stotypes.BlobRef, *stotypes.Blob) {
		blobRef, err := stotypes.BlobRefFromHex(blobRefSerialized)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, nil
		}

		tx, err := h.db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		blobMetadata, err := stodb.Read(tx).Blob(*blobRef)
		if err != nil {
			if err == blorm.ErrNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
				return nil, nil
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil, nil
			}
		}

		coll, err = stodb.Read(tx).Collection(collId)
		if err != nil {
			// cannot be bothered with a blorm.ErrNotFound check here, since that shouldn't happen
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil, nil
		}

		return blobRef, blobMetadata
	}

	blobRef, blobMetadata := getBlobMetadata(mux.Vars(r)["ref"])
	if blobMetadata == nil {
		return // error was handled in getBlobMetadata()
	}

	bestVolumeId, err := h.conf.DiskAccess.BestVolumeId(blobMetadata.Volumes)
	if err != nil {
		http.Error(w, stotypes.ErrBlobNotAccessibleOnThisNode.Error(), http.StatusInternalServerError)
		return
	}

	file, err := h.conf.DiskAccess.Fetch(*blobRef, coll.EncryptionKeys, bestVolumeId)
	if err != nil {
		if os.IsNotExist(err) {
			// should not happen, because metadata said that we should have this blob
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if _, err := io.Copy(w, file); err != nil {
		// FIXME: shouldn't try to write headers if even one write went to ResponseWriter
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handlers) UploadBlob(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	// we need a hint from the client of what the collection is, so we can resolve a
	// volume onto which the blob should be stored
	collectionId := r.URL.Query().Get("collection")

	// optimization to skip opportunistic compression if the client knows for sure the
	// content is not well compressible (video/audio/etc.)
	maybeCompressible := true
	if r.URL.Query().Get("maybe_compressible") != "" {
		var err error
		maybeCompressible, err = parseStringBool(r.URL.Query().Get("maybe_compressible"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	blobRef, err := stotypes.BlobRefFromHex(mux.Vars(r)["ref"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var volumeId int
	if err := h.db.View(func(tx *bolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(collectionId)
		if err != nil {
			return err
		}

		volumeId = coll.DesiredVolumes[0]

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.conf.DiskAccess.WriteBlob(volumeId, collectionId, *blobRef, r.Body, maybeCompressible); err != nil {
		// FIXME: some could be StatusBadRequest
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handlers) GetCollection(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	tx, err := h.db.Begin(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	coll, err := stodb.Read(tx).Collection(mux.Vars(r)["id"])
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	outJson(w, coll)
}

func (h *handlers) GetReconcilableItems(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.ReconciliationReport {
	tx, err := h.db.Begin(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	nonCompliantItems := []stoservertypes.ReconcilableItem{}

	max := 100

	totalItems := 0

	report := latestReconciliationReport
	if report != nil {
		totalItems = len(report.CollectionsWithNonCompliantPolicy)

		for idx, ctr := range report.CollectionsWithNonCompliantPolicy {
			if idx+1 >= max {
				break
			}

			coll, err := stodb.Read(tx).Collection(ctr.collectionId)
			if err != nil {
				panic(err)
			}

			path := []string{coll.Name}

			dirId := coll.Directory
			for dirId != "" {
				dir, err := stodb.Read(tx).Directory(dirId)
				if err != nil {
					panic(err)
				}

				path = append([]string{dir.Name}, path...)

				dirId = dir.Parent
			}

			presenceItems := []string{}

			fullReplicas := []int{}

			for volId, blobCount := range ctr.presence {
				if ctr.blobCount == blobCount {
					fullReplicas = append(fullReplicas, volId)
				}

				presenceItems = append(presenceItems, fmt.Sprintf("%d[%d blobs]", volId, blobCount))
			}

			nonCompliantItems = append(nonCompliantItems, stoservertypes.ReconcilableItem{
				CollectionId:    ctr.collectionId,
				Description:     strings.Join(path, " » "),
				TotalBlobs:      ctr.blobCount,
				DesiredReplicas: ctr.desiredReplicas,
				FullReplicas:    fullReplicas,
				Presence:        strings.Join(presenceItems, " "),
			})
		}
	}

	return &stoservertypes.ReconciliationReport{
		Items:      nonCompliantItems,
		TotalItems: totalItems,
	}
}

func (h *handlers) GenerateIds(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.GeneratedIds {
	return &stoservertypes.GeneratedIds{
		Changeset: stoutils.NewCollectionChangesetId(),
	}
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

func createDummyMiddlewares(conf *ServerConfig) httpauth.MiddlewareChainMap {
	return httpauth.MiddlewareChainMap{
		"public": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			return &httpauth.RequestContext{}
		},
		"authenticated": func(w http.ResponseWriter, r *http.Request) *httpauth.RequestContext {
			if !authenticate(conf, w, r) {
				return nil
			}

			return &httpauth.RequestContext{}
		},
	}
}

func getParentDirs(of stotypes.Directory, tx *bolt.Tx) ([]stotypes.Directory, error) {
	parentDirs := []stotypes.Directory{}

	current := &of
	var err error

	for current.Parent != "" {
		current, err = stodb.Read(tx).Directory(current.Parent)
		if err != nil {
			return nil, err
		}

		// reverse order
		parentDirs = append([]stotypes.Directory{*current}, parentDirs...)
	}

	return parentDirs, nil
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

func doesBlobExist(ref stotypes.BlobRef, db *bolt.DB) (bool, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	_, err = stodb.Read(tx).Blob(ref)
	if err == nil {
		return true, nil
	}
	if err == blorm.ErrNotFound {
		return false, nil
	}

	return false, err // unknown error
}

func getHealthCheckerGraph(db *bolt.DB, conf *ServerConfig) (stohealth.HealthChecker, error) {
	temps := []stohealth.HealthChecker{}
	smarts := []stohealth.HealthChecker{}
	replicationQueues := []stohealth.HealthChecker{}

	now := time.Now()

	if err := db.View(func(tx *bolt.Tx) error {
		return stodb.VolumeRepository.Each(func(record interface{}) error {
			vol := record.(*stotypes.Volume)

			replicationController, hasReplicationController := conf.ReplicationControllers[vol.ID]
			if hasReplicationController {
				replicationProgress := replicationController.Progress()

				if replicationProgress != 100 {
					replicationQueues = append(replicationQueues, stohealth.NewStaticHealthNode(
						vol.Label,
						stoservertypes.HealthStatusWarn,
						fmt.Sprintf("Progress at %d %%", replicationProgress)))
				} else {
					replicationQueues = append(replicationQueues, stohealth.NewStaticHealthNode(
						vol.Label,
						stoservertypes.HealthStatusPass,
						"Realtime"))
				}
			}

			if vol.SmartReport == "" {
				return nil
			}

			report := &stoservertypes.SmartReport{}
			if err := json.Unmarshal([]byte(vol.SmartReport), report); err != nil {
				return err
			}

			if report.Temperature != nil {
				temps = append(temps, stohealth.NewStaticHealthNode(
					vol.Label,
					stoservertypes.HealthStatusPass,
					fmt.Sprintf("%d °C", *report.Temperature)))
			}

			smartStatus := stoservertypes.HealthStatusPass
			if !report.Passed {
				smartStatus = stoservertypes.HealthStatusFail
			}

			smarts = append(smarts, stohealth.NewStaticHealthNode(
				vol.Label,
				smartStatus,
				fmt.Sprintf("Checked %s ago", duration.Humanize(now.Sub(report.Time)))))

			return nil
		}, tx)
	}); err != nil {
		return nil, err
	}

	return stohealth.NewHealthFolder(
		"Varasto",
		stohealth.NewLastSuccessfullBackup(db),
		stohealth.NewLastIntegrityVerificationJob(db),
		stohealth.NewHealthFolder(
			"Temperatures",
			temps...),
		stohealth.NewHealthFolder(
			"SMART",
			smarts...),
		stohealth.NewHealthFolder(
			"Replication queue",
			replicationQueues...)), nil
}
