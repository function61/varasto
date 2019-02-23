package bupserver

import (
	"github.com/asdine/storm"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/stateresolver"
	"github.com/function61/pi-security-module/pkg/f61ui"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

func defineUi(router *mux.Router, conf *ServerConfig, db *storm.DB) error {
	assetsPath := "/assets"

	publicFiles := http.FileServer(http.Dir("./public/"))

	router.HandleFunc("/", f61ui.IndexHtmlHandler(assetsPath))
	router.PathPrefix(assetsPath + "/").Handler(http.StripPrefix(assetsPath+"/", publicFiles))
	router.Handle("/favicon.ico", publicFiles)
	router.Handle("/robots.txt", publicFiles)

	return defineLegacyUi(router, conf, db)
}

func defineLegacyUi(router *mux.Router, conf *ServerConfig, db *storm.DB) error {
	router.HandleFunc("/collections/{collectionId}/rev/{changesetId}/dl", func(w http.ResponseWriter, r *http.Request) {
		fileKey := r.URL.Query().Get("file")

		tx, err := db.Begin(false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		coll, err := QueryWithTx(tx).Collection(mux.Vars(r)["collectionId"])
		if err != nil {
			if err == ErrDbRecordNotFound {
				http.Error(w, "collection not found", http.StatusNotFound)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		state, err := stateresolver.ComputeStateAt(*coll, mux.Vars(r)["changesetId"])
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
			Ref      buptypes.BlobRef
			VolumeId int
		}

		refAndVolumeIds := []RefAndVolumeId{}
		for _, refSerialized := range file.BlobRefs {
			ref, err := buptypes.BlobRefFromHex(refSerialized)
			if err != nil { // should not happen
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			blob, err := QueryWithTx(tx).Blob(*ref)
			if err != nil {
				if err == ErrDbRecordNotFound {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else {
					http.Error(w, "blob pointed to by file metadata not found", http.StatusInternalServerError)
					return
				}
			}

			volumeId, found := volumeManagerBestVolumeIdForBlob(blob.Volumes, conf)
			if !found {
				http.Error(w, buptypes.ErrBlobNotAccessibleOnThisNode.Error(), http.StatusInternalServerError)
				return
			}

			refAndVolumeIds = append(refAndVolumeIds, RefAndVolumeId{
				Ref:      *ref,
				VolumeId: volumeId,
			})
		}

		db.Rollback() // eagerly b/c the below operation is slow

		w.Header().Set("Content-Type", contentTypeForFilename(fileKey))

		for _, refAndVolumeId := range refAndVolumeIds {
			chunkStream, err := conf.VolumeDrivers[refAndVolumeId.VolumeId].Fetch(
				refAndVolumeId.Ref)
			panicIfError(err)

			if _, err := io.Copy(w, chunkStream); err != nil {
				panic(err)
			}

			chunkStream.Close()
		}
	})

	return nil
}
