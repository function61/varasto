package varastoserver

import (
	"encoding/json"
	"github.com/function61/varasto/pkg/sliceutil"
	"github.com/function61/varasto/pkg/varastotypes"
	"mime"
	"net/http"
	"path/filepath"
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

func collectionHasChangesetId(id string, coll *varastotypes.Collection) bool {
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

func contentTypeForFilename(path string) string {
	ext := filepath.Ext(path)

	// works with uppercase extensions as well
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return contentType
}
