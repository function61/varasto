package bupserver

import (
	"encoding/json"
	"github.com/function61/bup/pkg/buptypes"
	"net/http"
	"regexp"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

var bearerRe = regexp.MustCompile("^Bearer (.+)")

func authenticate(serverConfig ServerConfig, w http.ResponseWriter, r *http.Request) bool {
	match := bearerRe.FindStringSubmatch(r.Header.Get("Authorization"))

	if match != nil {
		if _, tokenAllowed := serverConfig.ClientsAuthTokens[match[1]]; tokenAllowed {
			return true
		}
	}

	http.Error(w, "missing or incorrect Authorization header", http.StatusForbidden)
	return false
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
