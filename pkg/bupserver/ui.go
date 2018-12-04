package bupserver

import (
	"github.com/asdine/storm"
	// "github.com/function61/bup/pkg/buptypes"
	"github.com/gorilla/mux"
	"net/http"
)

func defineUi(router *mux.Router, db *storm.DB) error {
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	return nil
}
