package varastoserver

import (
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/seasonepisodedetector"
	"github.com/function61/varasto/pkg/themoviedbapi"
	"github.com/function61/varasto/pkg/varastotypes"
	"go.etcd.io/bbolt"
	"strconv"
	"strings"
)

// this is for movies
func (c *cHandlers) CollectionPullMetadata(cmd *CollectionPullMetadata, ctx *command.Ctx) error {
	tmdb, err := c.themoviedbapiClient()
	if err != nil {
		return err
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		collection, err := QueryWithTx(tx).Collection(cmd.Collection)
		if err != nil {
			return err
		}

		info, err := tmdb.OpenMovieByImdbId(cmd.ForeignKey)
		if err != nil {
			return err
		}

		// store because we might lose detail when scrubbing name
		if collection.Name != info.OriginalTitle {
			collection.Metadata["previous_name"] = collection.Name
		}

		collection.Name = info.OriginalTitle

		collection.Metadata[MetadataTheMovieDbMovieId] = strconv.Itoa(int(info.Id))
		if info.ExternalIds.ImdbId != "" {
			collection.Metadata[MetadataImdbId] = info.ExternalIds.ImdbId
		}
		if info.Overview != "" {
			collection.Metadata[MetadataOverview] = info.Overview
		}
		if info.RuntimeMinutes != 0 {
			collection.Metadata[MetadataVideoRuntimeMins] = strconv.Itoa(info.RuntimeMinutes)
		}
		if info.RevenueDollars != 0 {
			collection.Metadata[MetadataVideoRevenueDollars] = strconv.Itoa(int(info.RevenueDollars))
		}
		if info.BackdropPath != "" {
			collection.Metadata[MetadataBackdrop] = themoviedbapi.ImagePath(info.BackdropPath, "original")
		}
		if info.ReleaseDate != "" {
			collection.Metadata[MetadataReleaseDate] = info.ReleaseDate
		}

		return CollectionRepository.Update(collection, tx)
	})
}

// directory holds a bunch of series
func (c *cHandlers) DirectoryPullMetadata(cmd *DirectoryPullMetadata, ctx *command.Ctx) error {
	tmdb, err := c.themoviedbapiClient()
	if err != nil {
		return err
	}

	tv, err := tmdb.OpenTvByImdbId(cmd.ForeignKey)
	if err != nil {
		return err
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		dir, err := QueryWithTx(tx).Directory(cmd.Directory)
		if err != nil {
			return err
		}

		dir.Metadata[MetadataTheMovieDbTvId] = fmt.Sprintf("%d", tv.Id)

		if tv.BackdropPath != "" {
			dir.Metadata[MetadataBackdrop] = themoviedbapi.ImagePath(tv.BackdropPath, "original")
		}

		if tv.Overview != "" {
			dir.Metadata[MetadataOverview] = tv.Overview
		}

		if tv.Homepage != "" {
			dir.Metadata[MetadataHomepage] = tv.Homepage
		}

		if tv.ExternalIds.ImdbId != "" {
			dir.Metadata[MetadataImdbId] = tv.ExternalIds.ImdbId
		}

		return DirectoryRepository.Update(dir, tx)
	})
}

// this is for serie episodes
func (c *cHandlers) CollectionRefreshMetadataAutomatically(cmd *CollectionRefreshMetadataAutomatically, ctx *command.Ctx) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		// Collection is validated as non-empty
		collIds := strings.Split(cmd.Collection, ",")

		firstColl, err := QueryWithTx(tx).Collection(collIds[0])
		if err != nil {
			return err
		}

		firstCollDirectory, err := QueryWithTx(tx).Directory(firstColl.Directory)
		if err != nil {
			return err
		}

		parentDirs, err := getParentDirs(*firstCollDirectory, tx)
		if err != nil {
			return err
		}

		theTvDbSeriesId := ""
		for _, parentDir := range parentDirs {
			theTvDbSeriesId = parentDir.Metadata[MetadataTheMovieDbTvId]
			if theTvDbSeriesId != "" {
				break
			}
		}
		if theTvDbSeriesId == "" {
			theTvDbSeriesId = firstCollDirectory.Metadata[MetadataTheMovieDbTvId] // one last try
		}
		if theTvDbSeriesId == "" {
			return fmt.Errorf("could not resolve %s for collection", MetadataTheMovieDbTvId)
		}

		uniqueSeasonNumbers := []int{}

		type episodeAndCollPair struct {
			seasonEpisode seasonepisodedetector.Result
			coll          *varastotypes.Collection
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
			coll, err := QueryWithTx(tx).Collection(collId)
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

		tmdb, err := c.themoviedbapiClient()
		if err != nil {
			return err
		}

		for _, seasonNumber := range uniqueSeasonNumbers {
			episodes, err := tmdb.GetSeasonEpisodes(seasonNumber, theTvDbSeriesId)
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

				if coll.Metadata == nil {
					panic("should not be after migration")
				}

				coll.Metadata[MetadataTheMovieDbTvId] = theTvDbSeriesId
				coll.Metadata[MetadataTheMovieDbTvEpisodeId] = fmt.Sprintf("%d", ep.Id)

				if ep.Name != "" {
					coll.Metadata[MetadataTitle] = ep.Name
				}
				if ep.AirDate != "" {
					coll.Metadata[MetadataReleaseDate] = ep.AirDate
				}
				if ep.Overview != "" {
					coll.Metadata[MetadataOverview] = ep.Overview
				}
				if ep.StillPath != "" {
					coll.Metadata[MetadataThumbnail] = themoviedbapi.ImagePath(ep.StillPath, "original")
				}

				if err := CollectionRepository.Update(coll, tx); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (c *cHandlers) themoviedbapiClient() (*themoviedbapi.Client, error) {
	if c.conf.File.TheMovieDbApiKey == "" {
		return nil, errors.New("TheMovieDbApiKey not defined")
	}

	return themoviedbapi.New(c.conf.File.TheMovieDbApiKey), nil
}
