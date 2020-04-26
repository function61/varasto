// IGDB (Internet Game Database) API
package igdbapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/function61/gokit/ezhttp"
)

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{apiKey}
}

func (c *Client) GameById(ctx context.Context, id string) (*Game, error) {
	query := fmt.Sprintf(
		"fields %s; where id = %s;",
		"age_ratings,aggregated_rating,aggregated_rating_count,alternative_names,artworks,bundles,category,collection,cover,created_at,dlcs,expansions,external_games,first_release_date,follows,franchise,franchises,game_engines,game_modes,genres,hypes,involved_companies,keywords,multiplayer_modes,name,parent_game,platforms,player_perspectives,popularity,pulse_count,rating,rating_count,release_dates,screenshots,similar_games,slug,standalone_expansions,status,storyline,summary,tags,themes,time_to_beat,total_rating,total_rating_count,updated_at,url,version_parent,version_title,videos,websites",
		id)

	res := []Game{}
	if _, err := ezhttp.Post(
		ctx,
		endpointV3("/games"),
		ezhttp.Header("Accept", "application/json"),
		ezhttp.Header("user-key", c.apiKey),
		ezhttp.SendBody(strings.NewReader(query), "application/x-www-form-urlencoded"),
		ezhttp.RespondsJson(&res, true),
	); err != nil {
		return nil, err
	}

	if len(res) != 1 {
		return nil, fmt.Errorf("expected 1; got %d", len(res))
	}

	return &res[0], nil
}

func (c *Client) GameYoutubeVideoIds(ctx context.Context, id string) ([]string, error) {
	query := fmt.Sprintf("fields name,video_id; where game=%s;", id)

	videos := []struct {
		Name    string `json:"name"`
		VideoID string `json:"video_id"`
	}{}

	if _, err := ezhttp.Post(
		ctx,
		endpointV3("/game_videos"),
		ezhttp.Header("Accept", "application/json"),
		ezhttp.Header("user-key", c.apiKey),
		ezhttp.SendBody(strings.NewReader(query), "application/x-www-form-urlencoded"),
		ezhttp.RespondsJson(&videos, true),
	); err != nil {
		return nil, err
	}

	youtubeIds := []string{}

	maybeYoutubeVideoId := func(id string) bool { return len(id) == len("dQw4w9WgXcQ") }

	for _, video := range videos {
		// docs mention "usually youtube", but there's no field to separate these, so
		// we'll have to do our best to filter out non-youtube videos.
		if !maybeYoutubeVideoId(video.VideoID) {
			continue
		}

		youtubeIds = append(youtubeIds, video.VideoID)
	}

	return youtubeIds, nil
}

func (c *Client) GameScreenshotUrls(ctx context.Context, id string) ([]string, error) {
	query := fmt.Sprintf("fields image_id,url; where game = %s;", id)

	screenshots := []struct { // NOTE: more fields available than what we specify here
		ImageId string `json:"image_id"`
		// the response deceptively also has "url", but that points to a so small thumbnail
		// image it must be for ants
	}{}
	if _, err := ezhttp.Post(
		ctx,
		endpointV3("/screenshots"),
		ezhttp.Header("Accept", "application/json"),
		ezhttp.Header("user-key", c.apiKey),
		ezhttp.SendBody(strings.NewReader(query), "application/x-www-form-urlencoded"),
		ezhttp.RespondsJson(&screenshots, true),
	); err != nil {
		return nil, err
	}

	urls := []string{}
	for _, screenshot := range screenshots {
		urls = append(urls, originalImageURL(screenshot.ImageId))
	}

	return urls, nil
}

func (c *Client) WebsitesByGameId(ctx context.Context, id string) ([]Website, error) {
	query := fmt.Sprintf("fields category,game,trusted,url; where game=%s;", id)

	websites := []Website{}

	if _, err := ezhttp.Post(
		ctx,
		endpointV3("/websites"),
		ezhttp.Header("Accept", "application/json"),
		ezhttp.Header("user-key", c.apiKey),
		ezhttp.SendBody(strings.NewReader(query), "application/x-www-form-urlencoded"),
		ezhttp.RespondsJson(&websites, true),
	); err != nil {
		return nil, err
	}

	return websites, nil
}

// this game's ID is:
// - "com.frogmind.badland" on Google Play
// - "269670" on Steam
// - etc.
func (c *Client) ExternalIdsByGameId(ctx context.Context, id string) (*ExternalIds, error) {
	websites, err := c.WebsitesByGameId(ctx, id)
	if err != nil {
		return nil, err
	}

	ids := &ExternalIds{}

	for _, website := range websites {
		if extract, found := extractorByCategory[website.Category]; found {
			if err := extract(website.Url, ids); err != nil {
				return nil, err
			}
		}
	}

	return ids, nil
}

func originalImageURL(imageId string) string {
	return fmt.Sprintf("https://images.igdb.com/igdb/image/upload/t_original/%s.jpg", imageId)
}

func endpointV3(path string) string {
	return "https://api-v3.igdb.com" + path
}
