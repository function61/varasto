package bupserver

import (
	"encoding/json"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/sliceutil"
	"net/http"
	"regexp"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

var bearerRe = regexp.MustCompile("^Bearer (.+)")

func authenticate(serverConfig *ServerConfig, w http.ResponseWriter, r *http.Request) bool {
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

func missingFromLeftHandSide(lhs []int, rhs []int) []int {
	missing := []int{}

	for _, item := range rhs {
		if !sliceutil.ContainsInt(lhs, item) {
			missing = append(missing, item)
		}
	}

	return missing
}
