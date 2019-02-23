package bupserver

import (
	"github.com/function61/pi-security-module/pkg/f61ui"
	"github.com/gorilla/mux"
	"net/http"
)

func defineUi(router *mux.Router) error {
	assetsPath := "/assets"

	publicFiles := http.FileServer(http.Dir("./public/"))

	router.HandleFunc("/", f61ui.IndexHtmlHandler(assetsPath))
	router.PathPrefix(assetsPath + "/").Handler(http.StripPrefix(assetsPath+"/", publicFiles))
	router.Handle("/favicon.ico", publicFiles)
	router.Handle("/robots.txt", publicFiles)

	return nil
}
