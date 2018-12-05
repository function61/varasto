package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/stateresolver"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
)

func defineUi(router *mux.Router, db *storm.DB) error {
	templates, err := template.New("templatecollection").ParseGlob("templates/*.html")
	if err != nil {
		return err
	}

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/collections", http.StatusFound)
	})

	router.HandleFunc("/collections", func(w http.ResponseWriter, r *http.Request) {
		colls := []buptypes.Collection{}
		panicIfError(db.All(&colls))

		templates.Lookup("collections.html").Execute(w, colls)
	})

	router.HandleFunc("/volumes-and-mounts", func(w http.ResponseWriter, r *http.Request) {
		volumes := []buptypes.Volume{}
		panicIfError(db.All(&volumes))

		volumeMounts := []buptypes.VolumeMount{}
		panicIfError(db.All(&volumeMounts))

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
		coll := buptypes.Collection{}
		if err := db.One("ID", collectionId, &coll); err != nil {
			if err != storm.ErrNotFound {
				panicIfError(err)
			}

			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if changesetId == "" {
			changesetId = coll.Head
		}

		state, err := stateresolver.ComputeStateAt(coll, changesetId)
		panicIfError(err)

		type TemplateData struct {
			ChangesetId string
			Collection  buptypes.Collection
			TotalSize   int64
			FileList    []buptypes.File
		}

		files := state.FileList()

		totalSize := int64(0)

		for _, file := range files {
			totalSize += file.Size
		}

		templates.Lookup("collection.html").Execute(w, &TemplateData{
			ChangesetId: state.ChangesetId,
			Collection:  coll,
			TotalSize:   totalSize,
			FileList:    files,
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
