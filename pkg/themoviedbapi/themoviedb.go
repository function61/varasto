// themoviedb.org ("TMDb") REST API client
package themoviedbapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"strconv"
)

type Movie struct {
	Id          int64 `json:"id"`
	ExternalIds struct {
		ImdbId string `json:"imdb_id"`
	} `json:"external_ids"`
	Title          string `json:"title"`
	OriginalTitle  string `json:"original_title"`
	Overview       string `json:"overview"`
	RuntimeMinutes int    `json:"runtime"`
	ReleaseDate    string `json:"release_date"` // yyyy-mm-dd
	RevenueDollars int64  `json:"revenue"`
	BackdropPath   string `json:"backdrop_path"`
}

type Tv struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	BackdropPath string `json:"backdrop_path"`
	PosterPath   string `json:"poster_path"`
	Homepage     string `json:"homepage"`
	ExternalIds  struct {
		ImdbId string `json:"imdb_id"`
	} `json:"external_ids"`
}

type Episode struct {
	Id            uint64 `json:"id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	AirDate       string `json:"air_date"` // yyyy-mm-dd
	StillPath     string `json:"still_path"`
}

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func (c *Client) OpenMovieByImdbId(imdbId string) (*Movie, error) {
	id, err := c.findMovieByImdbId(imdbId)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	res := &Movie{}
	if _, err := ezhttp.Get(
		ctx,
		endpoint("/movie/"+id+"?api_key="+c.apiKey+"&append_to_response=external_ids"),
		ezhttp.RespondsJson(res, true)); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) OpenTvByImdbId(imdbId string) (*Tv, error) {
	id, err := c.findTvByImdbId(imdbId)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	res := &Tv{}
	if _, err := ezhttp.Get(
		ctx,
		endpoint("/tv/"+id+"?api_key="+c.apiKey+"&append_to_response=external_ids"),
		ezhttp.RespondsJson(res, true)); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) GetSeasonEpisodes(seasonNumber int, tvId string) ([]Episode, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	res := struct {
		Episodes []Episode `json:"episodes"`
	}{}

	if _, err := ezhttp.Get(
		ctx,
		endpoint("/tv/"+tvId+"/season/"+strconv.Itoa(seasonNumber)+"?api_key="+c.apiKey),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return nil, err
	}

	return res.Episodes, nil
}

func (c *Client) findMovieByImdbId(imdbId string) (string, error) {
	id, isMovie, err := c.findMovieOrTvByImdbId(imdbId)
	if err != nil {
		return "", err
	}

	if !isMovie {
		return "", errors.New("is not movie")
	}

	return id, nil
}

func (c *Client) findTvByImdbId(imdbId string) (string, error) {
	id, isMovie, err := c.findMovieOrTvByImdbId(imdbId)
	if err != nil {
		return "", err
	}

	if isMovie {
		return "", errors.New("is not tv")
	}

	return id, nil
}

func (c *Client) findMovieOrTvByImdbId(imdbId string) (string, bool, error) {
	res := struct {
		MovieResults []struct {
			Id int64 `json:"id"`
		} `json:"movie_results"`
		TvResults []struct {
			Id int64 `json:"id"`
		} `json:"tv_results"`
	}{}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	if _, err := ezhttp.Get(
		ctx,
		endpoint("/find/"+imdbId+"?external_source=imdb_id&api_key="+c.apiKey),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return "", false, err
	}

	if len(res.MovieResults)+len(res.TvResults) != 1 {
		return "", false, errors.New("expecting exactly 1 result for TV or movie")
	}

	if len(res.MovieResults) == 1 {
		return strconv.Itoa(int(res.MovieResults[0].Id)), true, nil
	} else {
		return strconv.Itoa(int(res.TvResults[0].Id)), false, nil
	}
}

func endpoint(path string) string {
	return "https://api.themoviedb.org/3" + path
}

// width=original|w500|...
func ImagePath(path string, width string) string {
	return fmt.Sprintf("https://image.tmdb.org/t/p/%s%s", width, path)
}
