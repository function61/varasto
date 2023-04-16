// themoviedb.org ("TMDb") REST API client
package themoviedbapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/function61/gokit/ezhttp"
)

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func (c *Client) OpenMovieByImdbId(ctx context.Context, imdbId string) (*Movie, error) {
	id, err := c.findMovieByImdbId(ctx, imdbId)
	if err != nil {
		return nil, err
	}

	return c.OpenMovie(ctx, id)
}

func (c *Client) OpenMovie(ctx context.Context, id string) (*Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	res := &Movie{}
	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/movie/"+id+"?api_key="+c.apiKey+"&append_to_response=external_ids"),
		ezhttp.RespondsJson(res, true)); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) OpenTvByImdbId(ctx context.Context, imdbId string) (*Tv, error) {
	id, err := c.findTvByImdbId(ctx, imdbId)
	if err != nil {
		return nil, err
	}

	return c.OpenTv(ctx, id)
}

func (c *Client) OpenTv(ctx context.Context, id string) (*Tv, error) {
	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	res := &Tv{}
	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/tv/"+id+"?api_key="+c.apiKey+"&append_to_response=external_ids"),
		ezhttp.RespondsJson(res, true)); err != nil {
		return nil, err
	}

	return res, nil
}

// doesn't support returning external IDs, but the "get one episode" does.
func (c *Client) GetSeasonEpisodes(ctx context.Context, seasonNumber int, tvId string) ([]Episode, error) {
	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	res := struct {
		Episodes []Episode `json:"episodes"`
	}{}

	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/tv/"+tvId+"/season/"+strconv.Itoa(seasonNumber)+"?api_key="+c.apiKey),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return nil, err
	}

	return res.Episodes, nil
}

func (c *Client) GetEpisodeExternalIds(
	ctx context.Context,
	tvId string,
	seasonNumber int,
	episodeNumber int,
) (*ExternalIds, error) {
	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	res := &ExternalIds{}

	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/tv/"+tvId+"/season/"+strconv.Itoa(seasonNumber)+"/episode/"+strconv.Itoa(episodeNumber)+"/external_ids?api_key="+c.apiKey),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return nil, fmt.Errorf("GetEpisodeExternalIds: %w", err)
	}

	return res, nil
}

func (c *Client) MultiSearch(ctx context.Context, query string) ([]MultiSearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	type Response struct {
		Results []MultiSearchResult `json:"results"`
	}

	res := Response{}
	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/search/multi?api_key="+c.apiKey+"&query="+url.QueryEscape(query)),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return nil, err
	}

	return res.Results, nil
}

func (c *Client) MovieCredits(ctx context.Context, id string) (*Credits, error) {
	credits := &Credits{}
	_, err := ezhttp.Get(ctx, endpointV3("/movie/"+id+"/credits?api_key="+c.apiKey), ezhttp.RespondsJson(credits, true))
	return credits, err
}

func (c *Client) TVCredits(ctx context.Context, id string, season int, episode int) (*Credits, error) {
	credits := &Credits{}
	_, err := ezhttp.Get(ctx, endpointV3(fmt.Sprintf("/tv/%s/season/%d/episode/%d/credits?api_key=%s", id, season, episode, c.apiKey)), ezhttp.RespondsJson(credits, true))
	return credits, err
}

func (c *Client) findMovieByImdbId(ctx context.Context, imdbId string) (string, error) {
	return c.findMovieOrTvByImdbId(ctx, imdbId, true)
}

func (c *Client) findTvByImdbId(ctx context.Context, imdbId string) (string, error) {
	return c.findMovieOrTvByImdbId(ctx, imdbId, false)
}

func (c *Client) findMovieOrTvByImdbId(ctx context.Context, imdbId string, expectMovie bool) (string, error) {
	res := struct {
		MovieResults []struct {
			Id int64 `json:"id"`
		} `json:"movie_results"`
		TvResults []struct {
			Id int64 `json:"id"`
		} `json:"tv_results"`
	}{}

	ctx, cancel := context.WithTimeout(ctx, ezhttp.DefaultTimeout10s)
	defer cancel()

	if _, err := ezhttp.Get(
		ctx,
		endpointV3("/find/"+imdbId+"?external_source=imdb_id&api_key="+c.apiKey),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return "", err
	}

	// can't assert len(MovieResults) + len(TvResults) == 1 because tt0903747 (= Breaking
	// Bad TV series in IMDb) yiels both a movie and a TV series in TMDb
	if expectMovie {
		if len(res.MovieResults) != 1 {
			return "", fmt.Errorf(
				"expecting exactly 1 movie result; got=%d",
				len(res.MovieResults))
		}

		return strconv.Itoa(int(res.MovieResults[0].Id)), nil
	} else {
		if len(res.TvResults) != 1 {
			return "", fmt.Errorf(
				"expecting exactly 1 TV result; got=%d",
				len(res.TvResults))
		}

		return strconv.Itoa(int(res.TvResults[0].Id)), nil
	}
}

func endpointV3(path string) string {
	return "https://api.themoviedb.org/3" + path
}

// width=original|w500|...
func ImagePath(path string, width ImageSize) string {
	return fmt.Sprintf("https://image.tmdb.org/t/p/%s%s", width, path)
}
