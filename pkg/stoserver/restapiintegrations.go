package stoserver

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/function61/gokit/httpauth"
	"github.com/function61/varasto/pkg/gokitbp"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
	"go.etcd.io/bbolt"
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
	_ *httpauth.RequestContext,
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
				Id:          encodeTmdbRef(result.MediaType, fmt.Sprintf("%d", result.ID)),
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
				Id:          encodeTmdbRef(result.MediaType, fmt.Sprintf("%d", result.ID)),
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

	game, err := igdb.GameByID(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if game.URL == "" {
		http.Error(w, "game found but its URL not", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, game.URL, http.StatusFound)
}

func (h *handlers) TmdbCredits(rctx *httpauth.RequestContext, w http.ResponseWriter, r *http.Request) *[]stoservertypes.TmdbCredit {
	collectionID := r.URL.Query().Get("collection")

	tmdbPK, err := tmdbMovieOrTVEpisodePrimaryKeyForCollection(collectionID, h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	tmdb, err := themoviedbapiClient(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	credits, err := func() (*themoviedbapi.Credits, error) {
		if tmdbPK.movieID != "" {
			return tmdb.MovieCredits(r.Context(), tmdbPK.movieID)
		} else {
			return tmdb.TVCredits(r.Context(), tmdbPK.tvID, tmdbPK.season, tmdbPK.episode)
		}
	}()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	// transform to our own data structure (to insulate us from remote datatype changes)
	return gokitbp.Pointer(lo.Map(credits.Cast, func(cast themoviedbapi.CreditsCastItem, _ int) stoservertypes.TmdbCredit {
		if cast.ID == 0 { // verify assumption
			panic("no cast ID")
		}

		return stoservertypes.TmdbCredit{
			Name:      cast.Name,
			PersonURL: cast.ID.URL(),
			Character: cast.Character,
			ProfilePictureURL: func() *string {
				if cast.ProfilePath == nil {
					return nil
				} else {
					return gokitbp.Pointer(cast.ProfilePath.URL(themoviedbapi.ImageSizeW138AndH175Face))
				}
			}(),
		}
	}))
}

// a reference to a movie (simple id) or a TV episode (need: 1) TV show ID 2) season number 3) episode number)
// (even though they have individual IDs for the episodes, the API just doesn't seem to support referencing via it...)
type tmdbPrimaryKey struct {
	movieID string

	tvID    string
	season  int
	episode int
}

func tmdbMovieOrTVEpisodePrimaryKeyForCollection(collectionID string, db *bbolt.DB) (*tmdbPrimaryKey, error) {
	withErr := func(err error) (*tmdbPrimaryKey, error) { return nil, err }

	tx, err := db.Begin(false)
	if err != nil {
		return withErr(err)
	}
	defer func() { ignoreError(tx.Rollback()) }()

	coll, err := stodb.Read(tx).Collection(collectionID)
	if err != nil {
		return withErr(err)
	}

	if id, has := coll.Metadata[stoservertypes.MetadataTheMovieDbMovieId]; has {
		return &tmdbPrimaryKey{movieID: id}, nil
	}

	if id, has := coll.Metadata[stoservertypes.MetadataTheMovieDbTvId]; has {
		seasonEpisode := seasonepisodedetector.Detect(coll.Name)
		if seasonEpisode == nil {
			return withErr(fmt.Errorf("not able to detect season/episode from: %s", coll.Name))
		}

		return &tmdbPrimaryKey{tvID: id, season: gokitbp.Must(strconv.Atoi(seasonEpisode.Season)), episode: gokitbp.Must(strconv.Atoi(seasonEpisode.Episode))}, nil
	}

	return withErr(fmt.Errorf("coll not a movie and not a serie: %s", collectionID))
}
