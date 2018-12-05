package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/sliceutil"
	"github.com/function61/bup/pkg/stateresolver"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"path/filepath"
)

func defineUi(router *mux.Router, db *storm.DB) error {
	templates, err := template.New("templatecollection").Funcs(template.FuncMap{
		"basename": func(in string) string { return filepath.Base(in) },
	}).ParseGlob("templates/*.html")
	if err != nil {
		return err
	}

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/browse/root", http.StatusFound)
	})

	router.HandleFunc("/browse/{directoryId}", func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		dir := buptypes.Directory{}
		panicIfError(tx.One("ID", mux.Vars(r)["directoryId"], &dir))

		parentDirs, err := getParentDirs(dir, tx)
		panicIfError(err)

		subDirs := []buptypes.Directory{}
		if err := tx.Find("Parent", dir.ID, &subDirs); err != nil && err != storm.ErrNotFound {
			panic(err)
		}

		colls := []buptypes.Collection{}
		if err := tx.Find("Directory", dir.ID, &colls); err != nil && err != storm.ErrNotFound {
			panic(err)
		}

		templates.Lookup("browse.html").Execute(w, struct {
			Directory         buptypes.Directory
			ParentDirectories []buptypes.Directory
			SubDirectories    []buptypes.Directory
			Collections       []buptypes.Collection
		}{
			Directory:         dir,
			ParentDirectories: parentDirs,
			SubDirectories:    subDirs,
			Collections:       colls,
		})
	})

	router.HandleFunc("/volumes-and-mounts", func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		volumes := []buptypes.Volume{}
		panicIfError(tx.All(&volumes))

		volumeMounts := []buptypes.VolumeMount{}
		panicIfError(tx.All(&volumeMounts))

		type WrappedVolume struct {
			Volume       buptypes.Volume
			QuotaUsedPct int
		}

		wrappedVolumes := []WrappedVolume{}
		for _, vol := range volumes {
			wrappedVolumes = append(wrappedVolumes, WrappedVolume{
				Volume:       vol,
				QuotaUsedPct: int((vol.BlobSizeTotal * 100) / vol.Quota),
			})
		}

		templates.Lookup("volumes-and-mounts.html").Execute(w, struct {
			WrappedVolumes []WrappedVolume
			Mounts         []buptypes.VolumeMount
		}{
			WrappedVolumes: wrappedVolumes,
			Mounts:         volumeMounts,
		})
	})

	router.HandleFunc("/replicationpolicies", func(w http.ResponseWriter, r *http.Request) {
		replicationPolicies := []buptypes.ReplicationPolicy{}
		panicIfError(db.All(&replicationPolicies))

		templates.Lookup("replicationpolicies.html").Execute(w, replicationPolicies)
	})

	router.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes := []buptypes.Node{}
		panicIfError(db.All(&nodes))

		templates.Lookup("nodes.html").Execute(w, nodes)
	})

	router.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		clients := []buptypes.Client{}
		panicIfError(db.All(&clients))

		templates.Lookup("clients.html").Execute(w, clients)
	})

	serveCollectionAt := func(collectionId string, changesetId string, w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			path = "."
		}

		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		coll := buptypes.Collection{}
		if err := tx.One("ID", collectionId, &coll); err != nil {
			if err != storm.ErrNotFound {
				panicIfError(err)
			}

			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if changesetId == "" {
			changesetId = coll.Head
		}

		dir := buptypes.Directory{}
		panicIfError(tx.One("ID", coll.Directory, &dir))

		parentDirs, err := getParentDirs(dir, tx)
		panicIfError(err)

		state, err := stateresolver.ComputeStateAt(coll, changesetId)
		panicIfError(err)

		files := state.FileList()

		peekResult := stateresolver.DirPeek(files, path)
		// reverse these for UI's sake
		peekResult.ParentDirs = sliceutil.ReverseStringSlice(peekResult.ParentDirs)

		totalSize := int64(0)

		for _, file := range files {
			totalSize += file.Size
		}

		templates.Lookup("collection.html").Execute(w, struct {
			ChangesetId               string
			Collection                buptypes.Collection
			Directory                 buptypes.Directory
			ParentDirectories         []buptypes.Directory
			TotalSize                 int64
			FileList                  []buptypes.File
			SelectedDirectoryContents stateresolver.DirPeekResult
		}{
			ChangesetId:               state.ChangesetId,
			Collection:                coll,
			Directory:                 dir,
			ParentDirectories:         parentDirs,
			TotalSize:                 totalSize,
			FileList:                  files,
			SelectedDirectoryContents: *peekResult,
		})
	}

	router.HandleFunc("/collections/{collectionId}/rev/{changesetId}", func(w http.ResponseWriter, r *http.Request) {
		serveCollectionAt(mux.Vars(r)["collectionId"], mux.Vars(r)["changesetId"], w, r)
	})

	router.HandleFunc("/collections/{collectionId}", func(w http.ResponseWriter, r *http.Request) {
		serveCollectionAt(mux.Vars(r)["collectionId"], "", w, r) // head revision
	})

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	return nil
}

func getParentDirs(of buptypes.Directory, tx storm.Node) ([]buptypes.Directory, error) {
	parentDirs := []buptypes.Directory{}

	current := of
	for current.Parent != "" {
		if err := tx.One("ID", current.Parent, &current); err != nil {
			return nil, err
		}

		// reverse order
		parentDirs = append([]buptypes.Directory{current}, parentDirs...)
	}

	return parentDirs, nil
}
