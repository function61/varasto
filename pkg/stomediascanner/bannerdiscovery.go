package stomediascanner

import (
	"context"
	"fmt"
	"strconv"

	"github.com/function61/varasto/pkg/igdbapi"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
)

// uses various metadata databases to try to discover banner for the collection.
// NOTE: nil error doesn't guarantee that banner URL will be returned
func discoverBannerUrl(
	ctx context.Context,
	coll *stotypes.Collection,
	conf *stoclient.ClientConfig,
) (string, error) {
	theMovieDbApikey, err := fetchServerConfig(ctx, stoservertypes.CfgTheMovieDbApikey, conf)
	if err != nil {
		return "", err
	}

	igdbApikey, err := fetchServerConfig(ctx, stoservertypes.CfgIgdbApikey, conf)
	if err != nil {
		return "", err
	}

	var tmdb *themoviedbapi.Client
	if theMovieDbApikey != "" {
		tmdb = themoviedbapi.New(theMovieDbApikey)
	}

	var igdb *igdbapi.Client
	if igdbApikey != "" {
		igdb = igdbapi.New(igdbApikey)
	}

	tmdbMovieId, isMovie := coll.Metadata[stoservertypes.MetadataTheMovieDbMovieId]
	tmdbTvId, isTv := coll.Metadata[stoservertypes.MetadataTheMovieDbTvId]
	tmdbTvEpisodeId, isTvEpisode := coll.Metadata[stoservertypes.MetadataTheMovieDbTvEpisodeId]
	igdbId, isGame := coll.Metadata[stoservertypes.MetadataIgdbGameId]

	if isMovie && tmdb != nil {
		info, err := tmdb.OpenMovie(ctx, tmdbMovieId)
		if err != nil {
			return "", err // TODO: don't error out
		}

		if info.BackdropPath != "" {
			return themoviedbapi.ImagePath(info.BackdropPath, themoviedbapi.ImageSizeOriginal), nil
		}
	}

	if isTvEpisode && tmdb != nil {
		sed := seasonepisodedetector.Detect(coll.Name)
		if sed == nil {
			return "", fmt.Errorf("failed to detect season/episode: %s", coll.Name)
		}

		seasonNumber, _ := strconv.Atoi(sed.Season)

		// c'mon fuck this stupid way of identification
		episodes, err := tmdb.GetSeasonEpisodes(ctx, seasonNumber, coll.Metadata[stoservertypes.MetadataTheMovieDbTvId])
		if err != nil {
			return "", err
		}

		for _, episode := range episodes {
			if strconv.Itoa(int(episode.Id)) == tmdbTvEpisodeId && episode.StillPath != "" {
				return themoviedbapi.ImagePath(episode.StillPath, themoviedbapi.ImageSizeOriginal), nil
			}
		}
	}

	// episodes have both TV ID and episode ID set exclude episodes here as not to assign TV
	// backdrop to an episode
	if isTv && !isTvEpisode && tmdb != nil {
		info, err := tmdb.OpenTv(ctx, tmdbTvId)
		if err != nil {
			return "", err
		}

		if info.BackdropPath != "" {
			return themoviedbapi.ImagePath(info.BackdropPath, themoviedbapi.ImageSizeOriginal), nil
		}
	}

	if isGame && igdb != nil {
		screenshotUrls, err := igdb.GameScreenshotUrls(ctx, igdbId)
		if err != nil {
			return "", err
		}

		if len(screenshotUrls) > 0 {
			return screenshotUrls[0], nil
		}
	}

	return "", nil
}
