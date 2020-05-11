package stoserver

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
	"github.com/gorilla/mux"
)

func (h *handlers) SearchIgdb(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.MetadataIgdbGame {
	igdb, err := igdbClient(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	suggestions := []stoservertypes.MetadataIgdbGame{}

	// TODO: validate query non-empty
	games, err := igdb.SearchGames(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	for _, game := range games {
		var releaseYearPtr *int

		if game.FirstReleaseDate != nil {
			releaseYear := time.Time(*game.FirstReleaseDate).Year()
			releaseYearPtr = &releaseYear
		}

		suggestions = append(suggestions, stoservertypes.MetadataIgdbGame{
			Id:          strconv.Itoa(game.ID),
			Title:       game.Name,
			ReleaseYear: releaseYearPtr,
		})
	}

	return &suggestions
}

func (h *handlers) SearchTmdbMovies(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.TmdbSearchResult {
	return h.searchTmdbInternal(rctx, w, r, r.URL.Query().Get("q"), themoviedbapi.MediaTypeMovie)
}

func (h *handlers) SearchTmdbTv(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.TmdbSearchResult {
	return h.searchTmdbInternal(rctx, w, r, r.URL.Query().Get("q"), themoviedbapi.MediaTypeTv)
}

func (h *handlers) searchTmdbInternal(
	rctx *httpauth.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	query string,
	mediaType string,
) *[]stoservertypes.TmdbSearchResult {
	if query == "" {
		http.Error(w, "empty query", http.StatusBadRequest)
		return nil
	}

	tmdb, err := themoviedbapiClient(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	results, err := tmdb.MultiSearch(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	transformed := []stoservertypes.TmdbSearchResult{}

	for _, result := range results {
		if result.MediaType != mediaType {
			continue
		}

		switch result.MediaType {
		case themoviedbapi.MediaTypeMovie:
			var releaseYear *int
			if result.ReleaseDate != "" {
				x, err := strconv.Atoi(result.ReleaseDate[0:4])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return nil
				}

				releaseYear = &x
			}

			transformed = append(transformed, stoservertypes.TmdbSearchResult{
				Id:          encodeTmdbRef(result.MediaType, fmt.Sprintf("%d", result.Id)),
				Title:       result.Title,
				ReleaseYear: releaseYear,
			})
		case themoviedbapi.MediaTypeTv:
			var releaseYear *int
			if result.FirstAirDate != "" {
				x, err := strconv.Atoi(result.FirstAirDate[0:4])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return nil
				}

				releaseYear = &x
			}

			transformed = append(transformed, stoservertypes.TmdbSearchResult{
				Id:          encodeTmdbRef(result.MediaType, fmt.Sprintf("%d", result.Id)),
				Title:       result.Name,
				ReleaseYear: releaseYear,
			})
		default:
			// ignore
		}
	}

	return &transformed
}

// you can't make URLs by ID to IGDB without "slug" (which I don't think can be guaranteed
// to stay constant), so we have to use the API to fetch the current URL when user wants to
// enter the site
func (h *handlers) IgdbIntegrationRedir(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) {
	igdb, err := igdbClient(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	game, err := igdb.GameById(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if game.Url == "" {
		http.Error(w, "game found but its URL not", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, game.Url, http.StatusFound)
}
