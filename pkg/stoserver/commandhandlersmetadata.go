package stoserver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/igdbapi"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
	"go.etcd.io/bbolt"
)

// this is for movies
func (c *cHandlers) CollectionPullTmdbMetadata(cmd *stoservertypes.CollectionPullTmdbMetadata, ctx *command.Ctx) error {
	tmdb, err := themoviedbapiClient(c.db)
	if err != nil {
		return err
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		collection, err := stodb.Read(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		var info *themoviedbapi.Movie

		// check if tmdb reference
		typ, tmdbId, err := decodeTmdbRef(cmd.ForeignKey)
		if err != nil {
			return err
		}

		// it is TMDb id (movie or other type)
		if typ != "" {
			if typ != themoviedbapi.MediaTypeMovie {
				return fmt.Errorf("trying to pull movie metadata but given tmdb type: %s", typ)
			}

			info, err = tmdb.OpenMovie(ctx.Ctx, tmdbId)
			if err != nil {
				return err
			}
		} else { // not TMDb ID => IMDb ID then
			info, err = tmdb.OpenMovieByImdbId(ctx.Ctx, cmd.ForeignKey)
			if err != nil {
				return err
			}
		}

		if cmd.ScrubName {
			if err := maybeRename(collection, info.OriginalTitle, tx); err != nil {
				return err
			}
		}

		collection.Metadata[stoservertypes.MetadataTheMovieDbMovieId] = strconv.Itoa(int(info.Id))
		if info.ExternalIds.ImdbId != "" {
			collection.Metadata[stoservertypes.MetadataImdbId] = info.ExternalIds.ImdbId
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
		if info.BackdropPath != "" {
			collection.Metadata[stoservertypes.MetadataBackdrop] = themoviedbapi.ImagePath(info.BackdropPath, "original")
		}
		if info.ReleaseDate != "" {
			collection.Metadata[stoservertypes.MetadataReleaseDate] = info.ReleaseDate
		}

		return stodb.CollectionRepository.Update(collection, tx)
	})
}

// directory holds a bunch of series
func (c *cHandlers) DirectoryPullMetadata(cmd *stoservertypes.DirectoryPullMetadata, ctx *command.Ctx) error {
	tmdb, err := themoviedbapiClient(c.db)
	if err != nil {
		return err
	}

	tv, err := tmdb.OpenTvByImdbId(ctx.Ctx, cmd.ForeignKey)
	if err != nil {
		return err
	}

	return c.db.Update(func(tx *bbolt.Tx) error {
		dir, err := stodb.Read(tx).Directory(cmd.Directory)
		if err != nil {
			return err
		}

		dir.Metadata[stoservertypes.MetadataTheMovieDbTvId] = fmt.Sprintf("%d", tv.Id)

		if tv.BackdropPath != "" {
			dir.Metadata[stoservertypes.MetadataBackdrop] = themoviedbapi.ImagePath(tv.BackdropPath, "original")
		}

		if tv.Overview != "" {
			dir.Metadata[stoservertypes.MetadataOverview] = tv.Overview
		}

		if tv.Homepage != "" {
			dir.Metadata[stoservertypes.MetadataHomepage] = tv.Homepage
		}

		if tv.ExternalIds.ImdbId != "" {
			dir.Metadata[stoservertypes.MetadataImdbId] = tv.ExternalIds.ImdbId
		}

		return stodb.DirectoryRepository.Update(dir, tx)
	})
}

// this is for serie episodes
func (c *cHandlers) CollectionRefreshMetadataAutomatically(cmd *stoservertypes.CollectionRefreshMetadataAutomatically, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		// Collection is validated as non-empty
		collIds := strings.Split(cmd.Collection, ",")

		firstColl, err := stodb.Read(tx).Collection(collIds[0])
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

		tmdbTvId := ""
		for _, parentDir := range parentDirs {
			tmdbTvId = parentDir.Metadata[stoservertypes.MetadataTheMovieDbTvId]
			if tmdbTvId != "" {
				break
			}
		}
		if tmdbTvId == "" {
			tmdbTvId = firstCollDirectory.Metadata[stoservertypes.MetadataTheMovieDbTvId] // one last try
		}
		if tmdbTvId == "" {
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

		for _, collId := range collIds {
			coll, err := stodb.Read(tx).Collection(collId)
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

		tmdb, err := themoviedbapiClient(c.db)
		if err != nil {
			return err
		}

		for _, seasonNumber := range uniqueSeasonNumbers {
			episodes, err := tmdb.GetSeasonEpisodes(ctx.Ctx, seasonNumber, tmdbTvId)
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

				if _, hasImdbId := coll.Metadata[stoservertypes.MetadataImdbId]; !hasImdbId {
					// unfortunately the batch GetSeasonEpisodes() can not be made to return
					// per-episode IMDB IDs in a single call
					externalIds, err := tmdb.GetEpisodeExternalIds(
						ctx.Ctx,
						tmdbTvId,
						ep.SeasonNumber,
						ep.EpisodeNumber)
					if err != nil {
						return err
					}

					if externalIds.ImdbId != "" {
						coll.Metadata[stoservertypes.MetadataImdbId] = externalIds.ImdbId
					}
				}

				coll.Metadata[stoservertypes.MetadataTheMovieDbTvId] = tmdbTvId
				coll.Metadata[stoservertypes.MetadataTheMovieDbTvEpisodeId] = fmt.Sprintf("%d", ep.Id)

				if ep.Name != "" {
					coll.Metadata[stoservertypes.MetadataTitle] = ep.Name
				}
				if ep.AirDate != "" {
					coll.Metadata[stoservertypes.MetadataReleaseDate] = ep.AirDate
				}
				if ep.Overview != "" {
					coll.Metadata[stoservertypes.MetadataOverview] = ep.Overview
				}
				if ep.StillPath != "" {
					coll.Metadata[stoservertypes.MetadataThumbnail] = themoviedbapi.ImagePath(
						ep.StillPath,
						"original")
				}

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

	igdbId := cmd.ForeignKey

	gameDetails, err := igdb.GameById(ctx.Ctx, igdbId)
	if err != nil {
		return err
	}

	screenshotUrls, err := igdb.GameScreenshotUrls(ctx.Ctx, igdbId)
	if err != nil {
		return err
	}

	youtubeVideoIds, err := igdb.GameYoutubeVideoIds(ctx.Ctx, igdbId)
	if err != nil {
		return err
	}

	externalIds, err := igdb.ExternalIdsByGameId(ctx.Ctx, igdbId)
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

		if externalIds.SteamId != nil {
			coll.Metadata[stoservertypes.MetadataSteamAppId] = *externalIds.SteamId
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

		if externalIds.GooglePlayAppId != nil {
			coll.Metadata[stoservertypes.MetadataGooglePlayApp] = *externalIds.GooglePlayAppId
		}

		if externalIds.AppleAppStoreAppId != nil {
			coll.Metadata[stoservertypes.MetadataAppleAppStoreApp] = *externalIds.AppleAppStoreAppId
		}

		if len(youtubeVideoIds) > 0 {
			// don't replace if it already has
			if _, has := coll.Metadata[stoservertypes.MetadataYoutubeId]; !has {
				// only link the first
				coll.Metadata[stoservertypes.MetadataYoutubeId] = youtubeVideoIds[0]
			}
		}

		if len(screenshotUrls) > 0 {
			coll.Metadata[stoservertypes.MetadataBackdrop] = screenshotUrls[0]
		}

		return stodb.CollectionRepository.Update(coll, tx)
	})
}

func (c *cHandlers) ConfigSetTheMovieDbApikey(cmd *stoservertypes.ConfigSetTheMovieDbApikey, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		if cmd.Apikey != "" { // allow clearing this without testing
			// validate the API key by trying to use the API
			client := themoviedbapi.New(cmd.Apikey)
			_, err := client.OpenMovieByImdbId(ctx.Ctx, "tt1226229") // one of my fav movies, way underrated :)
			if err != nil {
				return fmt.Errorf("failed validating API key: %w", err)
			}
		}

		return stodb.CfgTheMovieDbApikey.Set(cmd.Apikey, tx)
	})
}

func (c *cHandlers) ConfigSetIgdbApikey(cmd *stoservertypes.ConfigSetIgdbApikey, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		if cmd.Apikey != "" { // allow clearing this without testing
			if _, err := igdbapi.New(cmd.Apikey).GameById(ctx.Ctx, "20025"); err != nil {
				return fmt.Errorf("failed validating API key: %w", err)
			}
		}

		return stodb.CfgIgdbApikey.Set(cmd.Apikey, tx)
	})
}

func themoviedbapiClient(db *bbolt.DB) (*themoviedbapi.Client, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	apikey, err := stodb.CfgTheMovieDbApikey.GetRequired(tx)
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

	apikey, err := stodb.CfgIgdbApikey.GetRequired(tx)
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
	if _, hasPreviousName := coll.Metadata["previous_name"]; !hasPreviousName {
		coll.Metadata["previous_name"] = coll.Name
	}

	coll.Name = scrubbedName

	return validateUniqueNameWithinSiblings(coll.Directory, coll.Name, tx)
}
