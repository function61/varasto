package stomediascanner

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/igdbapi"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/themoviedbapi"
)

// uses various metadata databases to try to discover banner for the collection.
// NOTE: nil error doesn't guarantee that banner URL will be returned
func discoverBannerURL(
	ctx context.Context,
	coll *stotypes.Collection,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) (string, error) {
	tmdb, igdb, err := getCachedClients(ctx, conf, logl)
	if err != nil {
		return "", err
	}

	tmdbMovieID, isMovie := coll.Metadata[stoservertypes.MetadataTheMovieDbMovieId]
	tmdbTvID, isTv := coll.Metadata[stoservertypes.MetadataTheMovieDbTvId]
	tmdbTvEpisodeID, isTvEpisode := coll.Metadata[stoservertypes.MetadataTheMovieDbTvEpisodeId]
	igdbID, isGame := coll.Metadata[stoservertypes.MetadataIgdbGameId]

	if isMovie && tmdb != nil {
		info, err := tmdb.OpenMovie(ctx, tmdbMovieID)
		if err != nil {
			return "", err // TODO: don't error out
		}

		if info.BackdropPath != nil {
			return info.BackdropPath.URL(themoviedbapi.ImageSizeOriginal), nil
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
			if strconv.Itoa(int(episode.ID)) == tmdbTvEpisodeID && episode.StillPath != nil {
				return episode.StillPath.URL(themoviedbapi.ImageSizeOriginal), nil
			}
		}
	}

	// episodes have both TV ID and episode ID set exclude episodes here as not to assign TV
	// backdrop to an episode
	if isTv && !isTvEpisode && tmdb != nil {
		info, err := tmdb.OpenTv(ctx, tmdbTvID)
		if err != nil {
			return "", err
		}

		if info.BackdropPath != nil {
			return info.BackdropPath.URL(themoviedbapi.ImageSizeOriginal), nil
		}
	}

	if isGame && igdb != nil {
		screenshotUrls, err := igdb.GameScreenshotUrls(ctx, igdbID)
		if err != nil {
			return "", err
		}

		if len(screenshotUrls) > 0 {
			return screenshotUrls[0], nil
		}

		// no screenshots - try to get a cover instead

		coverUrls, err := igdb.GameCoverURLs(ctx, igdbID)
		if err != nil {
			return "", err
		}

		if len(coverUrls) > 0 {
			return coverUrls[0], nil
		}
	}

	return "", nil
}

// zero value is useful for initial "cached entry expired" situation
type cachedClients struct {
	ttl  time.Time
	tmdb *themoviedbapi.Client
	igdb *igdbapi.Client
}

var (
	clientCache   = &cachedClients{}
	clientCacheMu = sync.Mutex{}
)

// caches returned clients for 15 seconds (along with their fetched API keys)
func getCachedClients(
	ctx context.Context,
	conf *stoclient.ClientConfig,
	logl *logex.Leveled,
) (*themoviedbapi.Client, *igdbapi.Client, error) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if clientCache.ttl.Before(time.Now()) {
		logl.Debug.Println("refreshing API client cache")

		theMovieDBApikey, err := fetchServerConfig(ctx, stoservertypes.CfgTheMovieDbApikey, conf)
		if err != nil {
			return nil, nil, err
		}

		igdbApikey, err := fetchServerConfig(ctx, stoservertypes.CfgIgdbApikey, conf)
		if err != nil {
			return nil, nil, err
		}

		clientCache = &cachedClients{
			ttl: time.Now().Add(15 * time.Second),
		}

		if theMovieDBApikey != "" {
			clientCache.tmdb = themoviedbapi.New(theMovieDBApikey)
		}

		if igdbApikey != "" {
			clientCache.igdb = igdbapi.New(igdbApikey)
		}
	}

	return clientCache.tmdb, clientCache.igdb, nil
}
