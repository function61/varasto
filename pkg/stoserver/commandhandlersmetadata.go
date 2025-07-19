package stoserver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/igdbapi"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/stomediascanner/stomediascantypes"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stokeystore"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) CollectionTriggerMediaScan(cmd *stoservertypes.CollectionTriggerMediaScan, ctx *command.Ctx) error {
	scanner := stomediascantypes.NewRestClientUrlBuilder("https://localhost")

	mode := ""
	if cmd.AllowDestructive {
		mode = "a"
	}

	_, err := ezhttp.Get(ctx.Ctx, scanner.TriggerScan(cmd.Collection, mode), ezhttp.Client(ezhttp.InsecureTlsClient))
	return err
}

// this is for movies
func (c *cHandlers) CollectionPullTmdbMetadata(cmd *stoservertypes.CollectionPullTmdbMetadata, ctx *command.Ctx) error {
	tmdb, err := themoviedbapiClient(c.db)
	if err != nil {
		return err
	}

	var info *themoviedbapi.Movie

	// check if tmdb reference
	typ, tmdbID, err := decodeTmdbRef(cmd.ForeignKey)
	if err != nil {
		return err
	}

	// it is TMDb id (movie or other type)
	if typ != "" {
		if typ != themoviedbapi.MediaTypeMovie {
			return fmt.Errorf("trying to pull movie metadata but given tmdb type: %s", typ)
		}

		info, err = tmdb.OpenMovie(ctx.Ctx, tmdbID)
		if err != nil {
			return err
		}
	} else { // not TMDb ID => IMDb ID then
		info, err = tmdb.OpenMovieByImdbID(ctx.Ctx, cmd.ForeignKey)
		if err != nil {
			return err
		}
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		collection, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		if cmd.ScrubName {
			if err := maybeRename(collection, info.OriginalTitle, tx); err != nil {
				return err
			}
		}

		collection.Metadata[stoservertypes.MetadataTheMovieDbMovieId] = strconv.Itoa(int(info.ID))
		if info.ExternalIds.ImdbID != "" {
			collection.Metadata[stoservertypes.MetadataImdbId] = info.ExternalIds.ImdbID
		}
		if info.Overview != "" {
			collection.Metadata[stoservertypes.MetadataOverview] = info.Overview
		}
		if info.RuntimeMinutes != 0 {
			collection.Metadata[stoservertypes.MetadataVideoRuntimeMins] = strconv.Itoa(info.RuntimeMinutes)
		}
		if info.RevenueDollars != 0 {
			collection.Metadata[stoservertypes.MetadataVideoRevenueDollars] = strconv.Itoa(int(info.RevenueDollars))
		}
		if info.ReleaseDate != "" {
			collection.Metadata[stoservertypes.MetadataReleaseDate] = info.ReleaseDate
		}

		collection.BumpGlobalVersion()

		return stodb.CollectionRepository.Update(collection, tx)
	})
}

// directory holds a bunch of series
func (c *cHandlers) DirectoryPullTmdbMetadata(cmd *stoservertypes.DirectoryPullTmdbMetadata, ctx *command.Ctx) error {
	tmdb, err := themoviedbapiClient(c.db)
	if err != nil {
		return err
	}

	// check if tmdb reference
	typ, tmdbID, err := decodeTmdbRef(cmd.ForeignKey)
	if err != nil {
		return err
	}

	var tv *themoviedbapi.Tv

	// it is TMDb id (movie or other type)
	if typ != "" {
		if typ != themoviedbapi.MediaTypeTv {
			return fmt.Errorf("trying to pull TV show metadata but got type: %s", typ)
		}

		tv, err = tmdb.OpenTv(ctx.Ctx, tmdbID)
		if err != nil {
			return err
		}
	} else { // not TMDb ID => IMDb ID then
		tv, err = tmdb.OpenTvByImdbID(ctx.Ctx, cmd.ForeignKey)
		if err != nil {
			return err
		}
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Directory)
		if err != nil {
			return err
		}

		metaColl, err := metaCollForDir(dir, tx, c.conf.KeyStore)
		if err != nil {
			return err
		}

		metaColl.Metadata[stoservertypes.MetadataTheMovieDbTvId] = fmt.Sprintf("%d", tv.ID)

		if tv.Overview != "" {
			metaColl.Metadata[stoservertypes.MetadataOverview] = tv.Overview
		}

		if tv.Homepage != "" {
			metaColl.Metadata[stoservertypes.MetadataHomepage] = tv.Homepage
		}

		if tv.ExternalIds.ImdbID != "" {
			metaColl.Metadata[stoservertypes.MetadataImdbId] = tv.ExternalIds.ImdbID
		}

		metaColl.BumpGlobalVersion()

		return stodb.CollectionRepository.Update(metaColl, tx)
	})
}

// this is for serie episodes
func (c *cHandlers) CollectionRefreshMetadataAutomatically(cmd *stoservertypes.CollectionRefreshMetadataAutomatically, ctx *command.Ctx) error {
	tmdb, err := themoviedbapiClient(c.db)
	if err != nil {
		return err
	}

	collIDs := *cmd.Collections

	if len(collIDs) == 0 {
		return nil // no-op
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		firstColl, err := stodb.Read(tx).Collection(collIDs[0])
		if err != nil {
			return err
		}

		firstCollDirectory, err := stodb.Read(tx).Directory(firstColl.Directory)
		if err != nil {
			return err
		}

		parentDirs, err := getParentDirs(*firstCollDirectory, tx)
		if err != nil {
			return err
		}

		tmdbTvID := ""
		for _, parentDir := range append(parentDirs, *firstCollDirectory) {
			if parentDir.MetaCollection != "" {
				metaColl, err := stodb.Read(tx).Collection(parentDir.MetaCollection)
				if err != nil {
					return err
				}

				tmdbTvID = metaColl.Metadata[stoservertypes.MetadataTheMovieDbTvId]
				if tmdbTvID != "" {
					break
				}
			}

		}
		if tmdbTvID == "" {
			return fmt.Errorf("could not resolve %s for collection", stoservertypes.MetadataTheMovieDbTvId)
		}

		uniqueSeasonNumbers := []int{}

		type episodeAndCollPair struct {
			seasonEpisode seasonepisodedetector.Result
			coll          *stotypes.Collection
		}

		pairs := []episodeAndCollPair{}

		findPair := func(seasonEpisode seasonepisodedetector.Result) *episodeAndCollPair {
			for _, pair := range pairs {
				if seasonEpisode.LaxEqual(pair.seasonEpisode) {
					return &pair
				}
			}

			return nil
		}

		for _, collID := range collIDs {
			coll, err := stodb.Read(tx).Collection(collID)
			if err != nil {
				return err
			}

			if coll.Directory != firstColl.Directory {
				return errors.New("all input collections must be siblings in the directory hierarchy")
			}

			seasonEpisode := seasonepisodedetector.Detect(coll.Name)
			if seasonEpisode == nil {
				continue
			}

			pairs = append(pairs, episodeAndCollPair{*seasonEpisode, coll})

			seasonNumber, err := strconv.Atoi(seasonEpisode.Season)
			if err != nil {
				return err // should not happen
			}

			if !sliceutil.ContainsInt(uniqueSeasonNumbers, seasonNumber) {
				uniqueSeasonNumbers = append(uniqueSeasonNumbers, seasonNumber)
			}
		}

		for _, seasonNumber := range uniqueSeasonNumbers {
			episodes, err := tmdb.GetSeasonEpisodes(ctx.Ctx, seasonNumber, tmdbTvID)
			if err != nil {
				return err
			}

			for _, ep := range episodes {
				seasonEpisode := seasonepisodedetector.Result{
					Season:  fmt.Sprintf("%d", ep.SeasonNumber),
					Episode: fmt.Sprintf("%d", ep.EpisodeNumber),
				}

				pair := findPair(seasonEpisode)
				if pair == nil {
					continue
				}

				coll := pair.coll

				if _, hasImdbID := coll.Metadata[stoservertypes.MetadataImdbId]; !hasImdbID {
					// unfortunately the batch GetSeasonEpisodes() can not be made to return
					// per-episode IMDB IDs in a single call
					externalIds, err := tmdb.GetEpisodeExternalIDs(
						ctx.Ctx,
						tmdbTvID,
						ep.SeasonNumber,
						ep.EpisodeNumber)
					if err != nil {
						return err
					}

					if externalIds.ImdbID != "" {
						coll.Metadata[stoservertypes.MetadataImdbId] = externalIds.ImdbID
					}
				}

				coll.Metadata[stoservertypes.MetadataTheMovieDbTvId] = tmdbTvID
				coll.Metadata[stoservertypes.MetadataTheMovieDbTvEpisodeId] = fmt.Sprintf("%d", ep.ID)

				if ep.Name != "" {
					coll.Metadata[stoservertypes.MetadataTitle] = ep.Name
				}
				if ep.AirDate != "" {
					coll.Metadata[stoservertypes.MetadataReleaseDate] = ep.AirDate
				}
				if ep.Overview != "" {
					coll.Metadata[stoservertypes.MetadataOverview] = ep.Overview
				}

				coll.BumpGlobalVersion()

				if err := stodb.CollectionRepository.Update(coll, tx); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (c *cHandlers) CollectionPullIgdbMetadata(cmd *stoservertypes.CollectionPullIgdbMetadata, ctx *command.Ctx) error {
	igdb, err := igdbClient(c.db)
	if err != nil {
		return err
	}

	igdbID := cmd.ForeignKey

	gameDetails, err := igdb.GameByID(ctx.Ctx, igdbID)
	if err != nil {
		return err
	}

	youtubeVideoIds, err := igdb.GameYoutubeVideoIDs(ctx.Ctx, igdbID)
	if err != nil {
		return err
	}

	externalIds, err := igdb.ExternalIDsByGameID(ctx.Ctx, igdbID)
	if err != nil {
		return err
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		coll, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		if cmd.ScrubName {
			if err := maybeRename(coll, gameDetails.Name, tx); err != nil {
				return err
			}
		}

		coll.Metadata[stoservertypes.MetadataIgdbGameId] = strconv.Itoa(gameDetails.ID)

		if gameDetails.Summary != "" {
			coll.Metadata[stoservertypes.MetadataOverview] = gameDetails.Summary
		}

		if gameDetails.FirstReleaseDate != nil {
			coll.Metadata[stoservertypes.MetadataReleaseDate] = time.Time(*gameDetails.FirstReleaseDate).UTC().Format("2006-01-02")
		}

		if externalIds.Official != nil {
			coll.Metadata[stoservertypes.MetadataHomepage] = *externalIds.Official
		}

		if externalIds.SteamID != nil {
			coll.Metadata[stoservertypes.MetadataSteamAppId] = *externalIds.SteamID
		}

		if externalIds.GogSlug != nil {
			coll.Metadata[stoservertypes.MetadataGogSlug] = *externalIds.GogSlug
		}

		if externalIds.RedditSlug != nil {
			coll.Metadata[stoservertypes.MetadataRedditSlug] = *externalIds.RedditSlug
		}

		if externalIds.EnglishWikipediaSlug != nil {
			coll.Metadata[stoservertypes.MetadataWikipediaSlug] = *externalIds.EnglishWikipediaSlug
		}

		if externalIds.GooglePlayAppID != nil {
			coll.Metadata[stoservertypes.MetadataGooglePlayApp] = *externalIds.GooglePlayAppID
		}

		if externalIds.AppleAppStoreAppID != nil {
			coll.Metadata[stoservertypes.MetadataAppleAppStoreApp] = *externalIds.AppleAppStoreAppID
		}

		if len(youtubeVideoIds) > 0 {
			// don't replace if it already has
			if _, has := coll.Metadata[stoservertypes.MetadataYoutubeId]; !has {
				// only link the first
				coll.Metadata[stoservertypes.MetadataYoutubeId] = youtubeVideoIds[0]
			}
		}

		coll.BumpGlobalVersion()

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) ConfigSetTheMovieDBApiKey(cmd *stoservertypes.ConfigSetTheMovieDBApiKey, ctx *command.Ctx) error {
	if cmd.Validation && cmd.Apikey != "" {
		if _, err := themoviedbapi.New(cmd.Apikey).OpenMovieByImdbID(ctx.Ctx, "tt1226229"); err != nil { // one of my fav movies, way underrated :)
			return fmt.Errorf("failed validating API key: %w", err)
		}
	}

	return c.setConfigValue(stodb.CfgTheMovieDBApikey, cmd.Apikey)
}

func (c *cHandlers) ConfigSetIgdbApikey(cmd *stoservertypes.ConfigSetIgdbApikey, ctx *command.Ctx) error {
	if cmd.Validation && cmd.Apikey != "" {
		if _, err := igdbapi.New(cmd.Apikey).GameByID(ctx.Ctx, "20025"); err != nil {
			return fmt.Errorf("failed validating API key: %w", err)
		}
	}

	return c.setConfigValue(stodb.CfgIgdbAPIkey, cmd.Apikey)
}

func themoviedbapiClient(db *bbolt.DB) (*themoviedbapi.Client, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	apikey, err := stodb.CfgTheMovieDBApikey.GetRequired(tx)
	if err != nil {
		return nil, err
	}

	return themoviedbapi.New(apikey), nil
}

func encodeTmdbRef(typ string, id string) string {
	return "tmdb:" + typ + ":" + id
}

func decodeTmdbRef(input string) (string, string, error) {
	if !strings.HasPrefix(input, "tmdb:") {
		return "", "", nil
	}

	components := strings.Split(input, ":")
	if len(components) != 3 {
		return "", "", fmt.Errorf("expected 3 components; got %d", len(components))
	}

	switch components[1] {
	case themoviedbapi.MediaTypeMovie:
		return themoviedbapi.MediaTypeMovie, components[2], nil
	case themoviedbapi.MediaTypeTv:
		return themoviedbapi.MediaTypeTv, components[2], nil
	default:
		return "", "", fmt.Errorf("unsupported tmdb type: %s", components[2])
	}
}

func igdbClient(db *bbolt.DB) (*igdbapi.Client, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	apikey, err := stodb.CfgIgdbAPIkey.GetRequired(tx)
	if err != nil {
		return nil, err
	}

	return igdbapi.New(apikey), nil
}

func maybeRename(coll *stotypes.Collection, scrubbedName string, tx *bbolt.Tx) error {
	if coll.Name == scrubbedName {
		return nil
	}
	// store as not to lose data when scrubbing name
	if _, hasPreviousName := coll.Metadata[stoservertypes.MetadataPreviousName]; !hasPreviousName {
		coll.Metadata[stoservertypes.MetadataPreviousName] = coll.Name
	}

	coll.Name = scrubbedName

	return validateUniqueNameWithinSiblings(coll.Directory, coll.Name, tx)
}

// - saves given directory, meta collection is created on-the-fly
// - possibly created meta collection is not saved
func metaCollForDir(dir *stotypes.Directory, tx *bbolt.Tx, keyStore *stokeystore.Store) (*stotypes.Collection, error) {
	var metaColl *stotypes.Collection

	if dir.MetaCollection != "" {
		return stodb.Read(tx).Collection(dir.MetaCollection)
	}

	metaColl, err := createCollection(stoservertypes.StoDirMetaName, dir.ID, keyStore, tx)
	if err != nil {
		return nil, err
	}

	dir.MetaCollection = metaColl.ID

	return metaColl, stodb.DirectoryRepository.Update(dir, tx)
}
