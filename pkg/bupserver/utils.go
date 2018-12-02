package bupserver

import (
	"crypto/subtle"
	"encoding/json"
	"github.com/function61/bup/pkg/buptypes"
	"net/http"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func authenticate(serverConfig ServerConfig, w http.ResponseWriter, r *http.Request) bool {
	auth := "Bearer " + serverConfig.ClientsAuthToken

	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(auth)) != 1 {
		http.Error(w, "missing or incorrect Authorization header", http.StatusForbidden)
		return false
	}

	return true
}

func outJson(w http.ResponseWriter, out interface{}) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(out)
}

func collectionHasChangesetId(id string, coll *buptypes.Collection) bool {
	for _, changeset := range coll.Changesets {
		if changeset.ID == id {
			return true
		}
	}

	return false
}

func missingFromLeftHandSide(lhs []string, rhs []string) []string {
	missing := []string{}

	for _, item := range rhs {
		if !contains(lhs, item) {
			missing = append(missing, item)
		}
	}

	return missing
}

type filterFn func(item string) bool

func filter(items []string, cb filterFn) []string {
	altered := []string{}

	for _, item := range items {
		if cb(item) {
			altered = append(altered, item)
		}
	}

	return altered
}

func contains(items []string, find string) bool {
	for _, item := range items {
		if item == find {
			return true
		}
	}

	return false
}
