package stoserver

// most of the REST endpoints for Varasto

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/function61/eventkit/guts"
	"github.com/function61/gokit/appuptime"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/httpauth"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/pi-security-module/pkg/httpserver/muxregistrator"
	"github.com/function61/varasto/pkg/blorm"
	"github.com/function61/varasto/pkg/duration"
	"github.com/function61/varasto/pkg/scheduler"
	"github.com/function61/varasto/pkg/stateresolver"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stodbimportexport"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stointegrityverifier"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"github.com/gorilla/mux"
	"github.com/minio/sha256-simd"
	"go.etcd.io/bbolt"
)

type handlers struct {
	db           *bbolt.DB
	conf         *ServerConfig
	ivController *stointegrityverifier.Controller
	logger       *log.Logger    // for sub-components
	logl         *logex.Leveled // for our logging
}

func defineRestApi(
	router *mux.Router,
	conf *ServerConfig,
	db *bbolt.DB,
	ivController *stointegrityverifier.Controller,
	mwares httpauth.MiddlewareChainMap,
	logger *log.Logger,
) error {
	var han stoservertypes.HttpHandlers = &handlers{
		db,
		conf,
		ivController,
		logger,
		logex.Levels(logger),
	}

	stoservertypes.RegisterRoutes(han, mwares, muxregistrator.New(router))

	return nil
}

func (h *handlers) GetDirectory(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.DirectoryOutput {
	httpErr := func(err error, errCode int) *stoservertypes.DirectoryOutput { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	dirId := mux.Vars(r)["id"]

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	dir, err := stodb.Read(tx).Directory(dirId)
	if err != nil {
		return httpErr(err, http.StatusNotFound)
	}

	parentDirsConverted, err := getParentDirsConverted(*dir, tx)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	dbColls, err := stodb.Read(tx).CollectionsByDirectory(dir.ID)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	collsWithMeta := []stoservertypes.CollectionSubsetWithMeta{}
	for _, dbColl := range dbColls {
		if dbColl.Name == stoservertypes.StoDirMetaName {
			continue
		}

		state, err := stateresolver.ComputeStateAtHead(dbColl)
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		collWithMeta, err := convertDbCollection(dbColl, nil, state)
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		collsWithMeta = append(collsWithMeta, *collWithMeta)
	}
	sort.Slice(collsWithMeta, func(i, j int) bool { return collsWithMeta[i].Collection.Name < collsWithMeta[j].Collection.Name })

	dbSubDirs, err := stodb.Read(tx).SubDirectories(dir.ID)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	subDirsWithMeta := []stoservertypes.DirectoryAndMeta{}
	for _, dbSubDir := range dbSubDirs {
		subDirWithMeta, err := newDirectoryAndMeta(convertDir(dbSubDir), tx)
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		subDirsWithMeta = append(subDirsWithMeta, *subDirWithMeta)
	}
	sort.Slice(subDirsWithMeta, func(i, j int) bool { return subDirsWithMeta[i].Directory.Name < subDirsWithMeta[j].Directory.Name })

	parentDirsAndMeta := []stoservertypes.DirectoryAndMeta{}
	for _, parent := range parentDirsConverted {
		parentDirAndMeta, err := newDirectoryAndMeta(parent, tx)
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		parentDirsAndMeta = append(parentDirsAndMeta, *parentDirAndMeta)
	}

	directoryAndMeta, err := newDirectoryAndMeta(convertDir(*dir), tx)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	return &stoservertypes.DirectoryOutput{
		Directory:      *directoryAndMeta,
		Parents:        parentDirsAndMeta,
		SubDirectories: subDirsWithMeta,
		Collections:    collsWithMeta,
	}
}

func (h *handlers) GetCollectiotAtRev(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.CollectionOutput {
	httpErr := func(err error, code int) *stoservertypes.CollectionOutput {
		http.Error(w, err.Error(), code)
		return nil
	}

	collectionId := mux.Vars(r)["id"]
	changesetId := mux.Vars(r)["rev"]
	pathBytes, err := base64.StdEncoding.DecodeString(mux.Vars(r)["path"])
	if err != nil {
		return httpErr(err, http.StatusBadRequest)
	}

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	coll, err := stodb.Read(tx).Collection(collectionId)
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return nil
		} else {
			return httpErr(err, http.StatusInternalServerError)
		}
	}

	if changesetId == stoservertypes.HeadRevisionId {
		changesetId = coll.Head
	}

	state, err := stateresolver.ComputeStateAt(*coll, changesetId)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	allFilesInRevision := state.FileList()

	// peek brings a subset of allFilesInRevision
	peekResult := stateresolver.DirPeek(allFilesInRevision, string(pathBytes))

	totalSize := int64(0)
	convertedFiles := []stoservertypes.File{}

	for _, file := range allFilesInRevision {
		totalSize += file.Size
	}

	for _, file := range peekResult.Files {
		convertedFiles = append(convertedFiles, convertFile(file))
	}

	changesetsConverted := []stoservertypes.ChangesetSubset{}

	for _, changeset := range coll.Changesets {
		changesetsConverted = append(changesetsConverted, stoservertypes.ChangesetSubset{
			Id:      changeset.ID,
			Parent:  changeset.Parent,
			Created: changeset.Created,
		})
	}

	collApi, err := convertDbCollection(*coll, changesetsConverted, state)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	return &stoservertypes.CollectionOutput{
		TotalSize: int(totalSize), // FIXME
		SelectedPathContents: stoservertypes.SelectedPathContents{
			Path:       peekResult.Path,
			Files:      convertedFiles,
			ParentDirs: peekResult.ParentDirs,
			SubDirs:    peekResult.SubDirs,
		},
		FileCount:          len(allFilesInRevision),
		ChangesetId:        changesetId,
		CollectionWithMeta: *collApi,
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
	defer func() { ignoreError(tx.Rollback()) }()

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

	if changesetId == stoservertypes.HeadRevisionId {
		changesetId = coll.Head
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

	// eagerly b/c the below operation (send to client) can be slow
	if err := tx.Rollback(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypeForFilename(fileKey))
	w.Header().Set("Content-Length", strconv.Itoa(int(file.Size)))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, path.Base(fileKey)))

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
		convert(h.conf.MediaScanner),
		convert(h.conf.FuseProjector),
	}
}

func (h *handlers) SearchVolumes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.Volume {
	query := strings.ToLower(r.URL.Query().Get("q"))

	return h.getVolumesInternal(rctx, w, r, func(vol stotypes.Volume) bool {
		return vol.Decommissioned == nil && strings.Contains(strings.ToLower(vol.Label), query)
	})
}

func (h *handlers) GetVolumes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.Volume {
	return h.getVolumesInternal(rctx, w, r, func(vol stotypes.Volume) bool {
		return vol.Decommissioned == nil
	})
}

func (h *handlers) GetDecommissionedVolumes(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.Volume {
	return h.getVolumesInternal(rctx, w, r, func(vol stotypes.Volume) bool {
		return vol.Decommissioned != nil
	})
}

func (h *handlers) getVolumesInternal(
	rctx *httpauth.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	filter func(vol stotypes.Volume) bool,
) *[]stoservertypes.Volume {
	volumes := []stoservertypes.Volume{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

	dbObjects := []stotypes.Volume{}
	panicIfError(stodb.VolumeRepository.Each(stodb.VolumeAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		if !filter(dbObject) {
			continue
		}

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
			Backend:dbObject.SmartBackend,
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

		var decommissioned *stoservertypes.VolumeDecommissioned
		if dbObject.Decommissioned != nil {
			decommissioned = &stoservertypes.VolumeDecommissioned{
				At:     *dbObject.Decommissioned,
				Reason: dbObject.DecommissionReason,
			}
		}

		volumes = append(volumes, stoservertypes.Volume{
			Id:             dbObject.ID,
			Uuid:           dbObject.UUID,
			Label:          dbObject.Label,
			Description:    dbObject.Description,
			Notes:          dbObject.Notes,
			SerialNumber:   dbObject.SerialNumber,
			Zone:           dbObject.Zone,
			Technology:     stoservertypes.VolumeTechnology(dbObject.Technology),
			Manufactured:   mfg,
			WarrantyEnds:   we,
			Topology:       topology,
			Smart:          smartAttrs,
			Quota:          int(dbObject.Quota), // FIXME: lossy conversions here
			BlobSizeTotal:  int(dbObject.BlobSizeTotal),
			BlobCount:      int(dbObject.BlobCount),
			Decommissioned: decommissioned,
		})
	}

	sort.Slice(volumes, func(i, j int) bool { return volumes[i].Label < volumes[j].Label })

	return &volumes
}

func (h *handlers) GetVolumeMounts(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.VolumeMount {
	ret := []stoservertypes.VolumeMount{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

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
	httpErr := func(err error, errCode int) *stoservertypes.BlobMetadata { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	ref, err := stotypes.BlobRefFromHex(mux.Vars(r)["ref"])
	if err != nil {
		return httpErr(err, http.StatusBadRequest)
	}

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	blob, err := stodb.Read(tx).Blob(*ref)
	if err != nil {
		if err == blorm.ErrNotFound {
			return httpErr(errors.New("blob not found"), http.StatusNotFound)
		} else {
			return httpErr(err, http.StatusInternalServerError)
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
		mux.Vars(r)["id"],
		stotypes.NewChangeset(
			changeset.ID,
			changeset.Parent,
			changeset.Created,
			toDbFiles(changeset.FilesCreated),
			toDbFiles(changeset.FilesUpdated),
			changeset.FilesDeleted),
		h.db,
		h.conf)

	// FIXME: add "produces" to here because commitChangesetInternal responds with updated collection
	if coll != nil {
		h.logl.Debug.Printf("committed %s to coll %s", changeset.ID, coll.ID)

		ignoreError(outJson(w, coll))
	}
}

// returns the updated collection struct or nil if errored (in this case http error is
// output automatically)
func commitChangesetInternal(
	w http.ResponseWriter,
	collectionId string,
	changeset stotypes.CollectionChangeset,
	db *bbolt.DB,
	serverConf *ServerConfig,
) *stotypes.Collection {
	httpErr := func(errStr string, errCode int) *stotypes.Collection { // shorthand
		http.Error(w, errStr, errCode)
		return nil
	}

	if !changeset.AnyChanges() {
		return httpErr("no changes in changeset", http.StatusBadRequest)
	}

	tx, errTxBegin := db.Begin(true)
	if errTxBegin != nil {
		return httpErr(errTxBegin.Error(), http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	coll, err := stodb.Read(tx).Collection(collectionId)
	if err != nil {
		return httpErr(err.Error(), http.StatusNotFound)
	}

	if collectionHasChangesetId(changeset.ID, coll) {
		return httpErr("changeset ID already in collection", http.StatusBadRequest)
	}

	if changeset.Parent != stotypes.NoParentId && !collectionHasChangesetId(changeset.Parent, coll) {
		return httpErr("parent changeset not found", http.StatusBadRequest)
	}

	if changeset.Parent != coll.Head {
		// TODO: force push or rebase support?
		return httpErr("commit does not target current head. would result in dangling heads!", http.StatusBadRequest)
	}

	replicationPolicy, err := stodb.Read(tx).ReplicationPolicy(coll.ReplicationPolicy)
	if err != nil {
		return httpErr(err.Error(), http.StatusInternalServerError)
	}

	createdAndUpdated := append(changeset.FilesCreated, changeset.FilesUpdated...)

	for _, file := range createdAndUpdated {
		for _, refHex := range file.BlobRefs {
			ref, err := stotypes.BlobRefFromHex(refHex)
			if err != nil {
				return httpErr(err.Error(), http.StatusBadRequest)
			}

			blob, err := stodb.Read(tx).Blob(*ref)
			if err != nil {
				return httpErr(fmt.Sprintf("blob %s not found", ref.AsHex()), http.StatusBadRequest)
			}

			// FIXME: if same changeset mentions same blob many times, we update the old blob
			// metadata many times due to the transaction reads not seeing uncommitted writes
			blob.Referenced = true
			blob.VolumesPendingReplication = missingFromLeftHandSide(
				blob.Volumes,
				replicationPolicy.DesiredVolumes)

			// blob got deduplicated from somewhere, and thus it uses a DEK that our collection
			// currently doesn't have a copy of?
			if stotypes.FindDekEnvelope(blob.EncryptionKeyId, coll.EncryptionKeys) == nil {
				env, err := copyAndReEncryptDekFromAnotherCollection(
					blob.EncryptionKeyId,
					extractKekPubKeyFingerprints(coll),
					tx,
					serverConf.KeyStore)
				if err != nil {
					return httpErr(err.Error(), http.StatusInternalServerError)
				}

				// inject copy of DEK re-encrypted for target collection's DEKs
				coll.EncryptionKeys = append(coll.EncryptionKeys, *env)

				// TODO: is this update required? we'll update coll later anyway..
				if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
					return httpErr(err.Error(), http.StatusInternalServerError)
				}
			}

			if err := stodb.BlobRepository.Update(blob, tx); err != nil {
				return httpErr(err.Error(), http.StatusInternalServerError)
			}
		}
	}

	// update head pointer & calc Created timestamp
	if err := appendAndValidateChangeset(changeset, coll); err != nil {
		return httpErr(err.Error(), http.StatusBadRequest)
	}

	if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
		return httpErr(err.Error(), http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		return httpErr(err.Error(), http.StatusInternalServerError)
	}

	return coll
}

func (h *handlers) SearchReplicationPolicies(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ReplicationPolicy {
	q := strings.ToLower(r.URL.Query().Get("q"))

	return h.getReplicationPoliciesInternal(rctx, w, r, func(policy stotypes.ReplicationPolicy) bool {
		return strings.Contains(strings.ToLower(policy.Name), q)
	})
}

func (h *handlers) GetReplicationPolicies(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ReplicationPolicy {
	return h.getReplicationPoliciesInternal(rctx, w, r, func(_ stotypes.ReplicationPolicy) bool { return true })
}

func (h *handlers) getReplicationPoliciesInternal(
	rctx *httpauth.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	filter func(stotypes.ReplicationPolicy) bool,
) *[]stoservertypes.ReplicationPolicy {
	policies := []stoservertypes.ReplicationPolicy{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

	dbObjects := []stotypes.ReplicationPolicy{}
	panicIfError(stodb.ReplicationPolicyRepository.Each(stodb.ReplicationPolicyAppender(&dbObjects), tx))

	for _, dbObject := range dbObjects {
		if !filter(dbObject) {
			continue
		}

		policies = append(policies, stoservertypes.ReplicationPolicy{
			Id:             dbObject.ID,
			Name:           dbObject.Name,
			MinZones:       dbObject.MinZones,
			DesiredVolumes: dbObject.DesiredVolumes,
		})
	}

	return &policies
}

func (h *handlers) GetReplicationPoliciesForDirectories(
	rctx *httpauth.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
) *[]stoservertypes.ReplicationPolicyForDirectory {
	rpfd := []stoservertypes.ReplicationPolicyForDirectory{}

	tx, rollback, err := readTx(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer rollback()

	// might not deserve an index?
	if err := stodb.DirectoryRepository.Each(func(record interface{}) error {
		dir := record.(*stotypes.Directory)

		if dir.ReplicationPolicy == "" {
			return nil
		}

		parents, err := getParentDirsConverted(*dir, tx)
		if err != nil {
			return err
		}

		rpfd = append(rpfd, stoservertypes.ReplicationPolicyForDirectory{
			Directory:        convertDir(*dir),
			DirectoryParents: parents,
		})

		return nil
	}, tx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return &rpfd
}

func (h *handlers) GetSchedulerJobs(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.SchedulerJob {
	fetchJobs := func() ([]stotypes.ScheduledJob, error) {
		tx, err := h.db.Begin(false)
		if err != nil {
			return nil, err
		}
		defer func() { ignoreError(tx.Rollback()) }()

		dbJobs := []stotypes.ScheduledJob{}

		if err := stodb.ScheduledJobRepository.Each(stodb.ScheduledJobAppender(&dbJobs), tx); err != nil {
			return nil, err
		}

		return dbJobs, nil
	}

	jobInSchedulerById := map[string]*scheduler.JobSpec{}

	for _, jobInScheduler := range h.conf.Scheduler.Snapshot() {
		jobInScheduler := jobInScheduler // pin
		jobInSchedulerById[jobInScheduler.Id] = &jobInScheduler
	}

	dbJobs, err := fetchJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	jobs := []stoservertypes.SchedulerJob{}

	for _, dbJob := range dbJobs {
		var lastRun *stoservertypes.SchedulerJobLastRun
		if dbJob.LastRun != nil {
			var errorStrPtr *string
			if dbJob.LastRun.Error != "" {
				errorStrPtr = &dbJob.LastRun.Error
			}

			lastRun = &stoservertypes.SchedulerJobLastRun{
				Error:    errorStrPtr,
				Started:  dbJob.LastRun.Started,
				Finished: dbJob.LastRun.Finished,
			}
		}

		running := false
		fromScheduler := jobInSchedulerById[dbJob.ID]
		if fromScheduler != nil {
			running = fromScheduler.Running
		}

		var nextRun *time.Time
		if !dbJob.NextRun.IsZero() {
			// need copy because ..
			nextRunCopy := dbJob.NextRun

			// .. otherwise we'd take an address of "dbJob.NextRun" which is mutable due
			// to how Go's range works
			nextRun = &nextRunCopy
		}

		jobs = append(jobs, stoservertypes.SchedulerJob{
			Id:          dbJob.ID,
			Description: dbJob.Description,
			Enabled:     dbJob.Enabled,
			Kind:        dbJob.Kind,
			Schedule:    dbJob.Schedule,
			Running:     running,
			NextRun:     nextRun,
			LastRun:     lastRun,
		})
	}

	return &jobs
}

func (h *handlers) GetReplicationStatuses(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ReplicationStatus {
	statuses := []stoservertypes.ReplicationStatus{}
	for volId, controller := range h.conf.ReplicationControllers {
		statuses = append(statuses, stoservertypes.ReplicationStatus{
			VolumeId: volId,
			Progress: controller.Progress(),
		})
	}

	return &statuses
}

func (h *handlers) GetKeyEncryptionKeys(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.KeyEncryptionKey {
	ret := []stoservertypes.KeyEncryptionKey{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

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
	httpErr := func(err error, errCode int) *[]stoservertypes.Node { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	ret := []stoservertypes.Node{}

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	dbObjects := []stotypes.Node{}
	if err := stodb.NodeRepository.Each(stodb.NodeAppender(&dbObjects), tx); err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	for _, dbObject := range dbObjects {
		cert, err := cryptoutil.ParsePemX509Certificate([]byte(dbObject.TlsCert))
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		publicKeyHumanReadableDescription, err := cryptoutil.PublicKeyHumanReadableDescription(cert.PublicKey)
		if err != nil {
			return httpErr(err, http.StatusInternalServerError)
		}

		ret = append(ret, stoservertypes.Node{
			Id:   dbObject.ID,
			Addr: dbObject.Addr,
			Name: dbObject.Name,
			TlsCert: stoservertypes.TlsCertDetails{
				Identity:           cryptoutil.Identity(*cert),
				Issuer:             cryptoutil.Issuer(*cert),
				PublicKeyAlgorithm: publicKeyHumanReadableDescription,
				NotAfter:           cert.NotAfter,
			},
		})
	}

	return &ret
}

func (h *handlers) GetApiKeys(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.ApiKey {
	ret := []stoservertypes.ApiKey{}

	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

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
	defer func() { ignoreError(tx.Rollback()) }()

	w.Header().Set("Content-Type", "text/plain")

	processFile := func(file *stotypes.File) {
		fmt.Fprintf(w, "%s %s\n", file.Sha256, file.Path)
	}

	panicIfError(stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		for _, changeset := range coll.Changesets {
			for _, file := range changeset.FilesCreated {
				file := file // pin
				processFile(&file)
			}

			for _, file := range changeset.FilesUpdated {
				file := file // pin
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
// 	$ curl -H "Authorization: Bearer $BUP_AUTHTOKEN" https://localhost/api/db/export
func (h *handlers) DatabaseExport(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	tx, err := h.db.Begin(false)
	panicIfError(err)
	defer func() { ignoreError(tx.Rollback()) }()

	panicIfError(stodbimportexport.Export(tx, w))
}

func (h *handlers) GetLogs(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]string {
	lines := h.conf.LogTail.Snapshot()
	return &lines
}

func (h *handlers) UploadFile(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.File {
	collectionId := mux.Vars(r)["id"]
	mtimeUnixMillis, err := strconv.ParseInt(r.URL.Query().Get("mtime"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	mtime := time.Unix(mtimeUnixMillis/1000, 0)

	// TODO: reuse the logic found in the client package?
	wholeFileHash := sha256.New()

	var volumeId int
	if err := h.db.View(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(collectionId)
		if err != nil {
			return err
		}

		replicationPolicy, err := stodb.Read(tx).ReplicationPolicy(coll.ReplicationPolicy)
		if err != nil {
			return err
		}

		volumeId = replicationPolicy.DesiredVolumes[0]

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
			// do not verify sha256, as we just calculated it (turns out it's really expensive)
			if err := h.conf.DiskAccess.WriteBlobNoVerify(
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

	tx, rollback, err := readTx(h.db)
	panicIfError(err)
	defer rollback()

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
	httpErr := func(err error, errCode int) *stoservertypes.ConfigValue { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	key := mux.Vars(r)["id"]

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	var val string
	switch key {
	case stoservertypes.CfgFuseServerBaseUrl:
		val, err = stodb.CfgFuseServerBaseUrl.GetOptional(tx)
	case stoservertypes.CfgTheMovieDbApikey:
		val, err = stodb.CfgTheMovieDbApikey.GetOptional(tx)
	case stoservertypes.CfgIgdbApikey:
		val, err = stodb.CfgIgdbApikey.GetOptional(tx)
	case stoservertypes.CfgNetworkShareBaseUrl:
		val, err = stodb.CfgNetworkShareBaseUrl.GetOptional(tx)
	case stoservertypes.CfgUbackupConfig:
		val, err = stodb.CfgUbackupConfig.GetOptional(tx)
	case stoservertypes.CfgGrafanaUrl:
		val, err = stodb.CfgGrafanaUrl.GetOptional(tx)
	case stoservertypes.CfgMediascannerState:
		val, err = stodb.CfgMediascannerState.GetOptional(tx)
	default:
		return httpErr(fmt.Errorf("unknown key: %s", key), http.StatusNotFound)
	}
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
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

	// TODO: add SchemaVersion (if not to UI, here anyway?)

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

func (h *handlers) DownloadUbackupStoredBackup(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id not given", http.StatusBadRequest)
		return
	}

	conf, err := ubConfigFromDb(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, path.Base(id)))

	if err := downloadBackup(id, w, *conf, h.logger); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handlers) GetUbackupStoredBackups(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.UbackupStoredBackup {
	conf, err := ubConfigFromDb(h.db)
	if _, is := err.(*stodb.ConfigRequiredError); is {
		return &[]stoservertypes.UbackupStoredBackup{}
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	backups, err := listUbackupStoredBackups(conf.Storage, h.logger)
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
		defer func() { ignoreError(tx.Rollback()) }()

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
		h.logl.Error.Printf("error copying blob to response: %v", err)
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
	if err := h.db.View(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(collectionId)
		if err != nil {
			return err
		}

		replicationPolicy, err := stodb.Read(tx).ReplicationPolicy(coll.ReplicationPolicy)
		if err != nil {
			return err
		}

		volumeId = replicationPolicy.DesiredVolumes[0]

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.conf.DiskAccess.WriteBlob(
		volumeId,
		collectionId,
		*blobRef,
		r.Body,
		maybeCompressible,
	); err != nil {
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
	defer func() { ignoreError(tx.Rollback()) }()

	coll, err := stodb.Read(tx).Collection(mux.Vars(r)["id"])
	if err != nil {
		if err == blorm.ErrNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	ignoreError(outJson(w, coll))
}

func (h *handlers) GetReconcilableItems(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *stoservertypes.ReconciliationReport {
	httpErr := func(err error, errCode int) *stoservertypes.ReconciliationReport { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	tx, err := h.db.Begin(false)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer func() { ignoreError(tx.Rollback()) }()

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
				return httpErr(err, http.StatusNotFound)
			}

			path := []string{coll.Name}

			dirId := coll.Directory
			for dirId != "" {
				dir, err := stodb.Read(tx).Directory(dirId)
				if err != nil {
					return httpErr(err, http.StatusNotFound)
				}

				path = append([]string{dir.Name}, path...)

				dirId = dir.Parent
			}

			replicaStatuses := []stoservertypes.ReconcilableItemReplicaStatus{}

			for volId, blobCount := range ctr.presence {
				replicaStatuses = append(replicaStatuses, stoservertypes.ReconcilableItemReplicaStatus{
					Volume:    volId,
					BlobCount: blobCount,
				})
			}

			sort.Slice(replicaStatuses, func(i, j int) bool { return replicaStatuses[i].Volume < replicaStatuses[j].Volume })

			nonCompliantItems = append(nonCompliantItems, stoservertypes.ReconcilableItem{
				CollectionId:        ctr.collectionId,
				Description:         strings.Join(path, " » "),
				TotalBlobs:          ctr.blobCount,
				DesiredReplicaCount: ctr.desiredReplicas,
				ReplicaStatuses:     replicaStatuses,
				ProblemRedundancy:   ctr.problemRedundancy,
				ProblemZoning:       ctr.problemZoning,
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

func getParentDirs(of stotypes.Directory, tx *bbolt.Tx) ([]stotypes.Directory, error) {
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

func doesBlobExist(ref stotypes.BlobRef, db *bbolt.DB) (bool, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return false, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	_, err = stodb.Read(tx).Blob(ref)
	if err == nil {
		return true, nil
	}
	if err == blorm.ErrNotFound {
		return false, nil
	}

	return false, err // unknown error
}

func getHealthCheckerGraph(db *bbolt.DB, conf *ServerConfig) (stohealth.HealthChecker, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	now := time.Now()

	volumesOverQuota := []string{}

	temps := []stohealth.HealthChecker{}
	smarts := []stohealth.HealthChecker{}
	replicationQueues := []stohealth.HealthChecker{}

	if err := stodb.VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)

		if vol.BlobSizeTotal > vol.Quota {
			volumesOverQuota = append(volumesOverQuota, vol.Label)
		}

		volReplicationHealth, err := healthVolReplication(vol, tx, conf)
		if err != nil {
			return err
		}
		if volReplicationHealth != nil {
			replicationQueues = append(replicationQueues, volReplicationHealth)
		}

		// even if we have report but SMART has been disabled afterwards, we shouldn't
		// use the report anymore b/c most likely it's outdated
		if vol.SmartReport == "" || vol.SmartId == "" {
			return nil
		}

		report := &stoservertypes.SmartReport{}
		if err := json.Unmarshal([]byte(vol.SmartReport), report); err != nil {
			return err
		}

		if report.Temperature != nil {
			temps = append(temps, stohealth.NewStaticHealthNode(
				vol.Label,
				temperatureToHealthStatus(*report.Temperature),
				fmt.Sprintf("%d °C", *report.Temperature),
				nil))
		}

		smarts = append(smarts, func() stohealth.HealthChecker {
			volHealth := staticHealthBuilder(vol.Label, nil)

			reportAge := now.Sub(report.Time)
			agoText := fmt.Sprintf("Checked %s ago", duration.Humanize(reportAge))

			if !report.Passed {
				return volHealth.Fail(agoText)
			}

			if reportAge > 24*time.Hour {
				return volHealth.Warn(fmt.Sprintf("%s (check too old)", agoText))
			}

			return volHealth.Pass(agoText)
		}())

		return nil
	}, tx); err != nil {
		return nil, err
	}

	checkers := []stohealth.HealthChecker{
		healthRunningLatestVersion(tx),
		healthNoFailedMounts(conf.FailedMountNames),
		serverCertHealth(
			conf.TlsCertificate.cert.NotAfter,
			time.Now()),
		quotaHealth(volumesOverQuota),
		healthNoReconciliationConflicts(),
		stohealth.NewLastIntegrityVerificationJob(db),
	}

	if len(temps) > 0 {
		checkers = append(checkers, stohealth.NewHealthFolder(
			"Temperatures",
			stoservertypes.HealthKindSmart.Ptr(),
			temps...))
	}

	if len(smarts) > 0 {
		checkers = append(checkers, stohealth.NewHealthFolder(
			"SMART diagnostics",
			stoservertypes.HealthKindSmart.Ptr(),
			smarts...))
	}

	checkers = append(checkers,
		healthSubsystems(conf.MediaScanner, conf.FuseProjector),
		healthForScheduledJobs(tx),
		stohealth.NewHealthFolder(
			"Replication queue",
			stoservertypes.HealthKindVolumeReplication.Ptr(),
			replicationQueues...))

	return stohealth.NewHealthFolder("Varasto", nil, checkers...), nil
}
