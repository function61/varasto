package stoserver

import (
	"encoding/binary"
	"net/http"
	"strconv"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func (h *handlers) CollectionChangefeed(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.CollectionChangefeedItem {
	//nolint:unparam // false positive for this pattern
	httpErr := func(err error, errCode int) *[]stoservertypes.CollectionChangefeedItem { // shorthand
		http.Error(w, err.Error(), errCode)
		return nil
	}

	cursor, err := func() (uint64, error) {
		after := r.URL.Query().Get("after")
		if after != "" {
			return strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
		} else {
			return 0, nil // just start from beginning
		}
	}()
	if err != nil {
		return httpErr(err, http.StatusBadRequest)
	}

	tx, rollback, err := readTx(h.db)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}
	defer rollback()

	// +1 so we exclude the items we already processed
	globalVersionStart := make([]byte, 8)
	binary.BigEndian.PutUint64(globalVersionStart, cursor+1)

	results := []stoservertypes.CollectionChangefeedItem{}
	err = stodb.CollectionsGlobalVersionIndex.Query(globalVersionStart, func(sortKey []byte, value []byte) error {
		globalVersion := binary.BigEndian.Uint64(sortKey)

		results = append(results, stoservertypes.CollectionChangefeedItem{
			Cursor:       strconv.FormatUint(globalVersion, 10),
			CollectionId: string(value),
		})

		if len(results) >= 50 {
			return stodb.StopIteration
		} else {
			return nil
		}
	}, tx)
	if err != nil {
		return httpErr(err, http.StatusInternalServerError)
	}

	return &results
}
