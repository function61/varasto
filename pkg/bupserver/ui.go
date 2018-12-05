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
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		colls := []buptypes.Collection{}
		panicIfError(tx.All(&colls))

		templates.Lookup("collections.html").Execute(w, colls)
	})

	router.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		volumes := []buptypes.Volume{}
		panicIfError(tx.All(&volumes))

		templates.Lookup("volumes.html").Execute(w, volumes)
	})

	router.HandleFunc("/replicationpolicies", func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		replicationPolicies := []buptypes.ReplicationPolicy{}
		panicIfError(tx.All(&replicationPolicies))

		templates.Lookup("replicationpolicies.html").Execute(w, replicationPolicies)
	})

	router.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin(false)
		panicIfError(err)
		defer tx.Rollback()

		nodes := []buptypes.Node{}
		panicIfError(tx.All(&nodes))

		templates.Lookup("nodes.html").Execute(w, nodes)
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
			FileList    []buptypes.File
		}

		templates.Lookup("collection.html").Execute(w, &TemplateData{
			ChangesetId: state.ChangesetId,
			Collection:  coll,
			FileList:    state.FileList(),
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
