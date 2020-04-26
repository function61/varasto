package igdbapi

import (
	"strconv"
	"time"
)

// https://api-docs.igdb.com/#website-enums
const (
	WebsiteOfficial  = 1
	WebsiteWikia     = 2
	WebsiteWikipedia = 3
	WebsiteFacebook  = 4
	WebsiteTwitter   = 5
	WebsiteTwitch    = 6
	WebsiteInstagram = 8
	WebsiteYoutube   = 9
	WebsiteIphone    = 10
	WebsiteIpad      = 11
	WebsiteAndroid   = 12
	WebsiteSteam     = 13
	WebsiteReddit    = 14
	WebsiteItch      = 15
	WebsiteEpicgames = 16
	WebsiteGog       = 17
)

type Game struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Summary          string    `json:"summary"`
	FirstReleaseDate *UnixTime `json:"first_release_date"`
	Url              string    `json:"url"`
	/* present, but not using / vetted these yet
	AggregatedRating      int    `json:"aggregated_rating"`
	AggregatedRatingCount int    `json:"aggregated_rating_count"`
	Category              int    `json:"category"`
	Cover                 int    `json:"cover"`
	CreatedAt             int    `json:"created_at"`
	ExternalGames         []int  `json:"external_games"`
	GameModes             []int  `json:"game_modes"`
	Genres                []int  `json:"genres"`
	InvolvedCompanies     []int  `json:"involved_companies"`
	Platforms             []int  `json:"platforms"`
	Popularity            int    `json:"popularity"`
	PulseCount            int    `json:"pulse_count"`
	Rating                int    `json:"rating"`
	RatingCount           int    `json:"rating_count"`
	ReleaseDates          []int  `json:"release_dates"`
	Screenshots           []int  `json:"screenshots"`
	SimilarGames          []int  `json:"similar_games"`
	Slug                  string `json:"slug"`
	Tags                  []int  `json:"tags"`
	Themes                []int  `json:"themes"`
	TotalRating           int    `json:"total_rating"`
	TotalRatingCount      int    `json:"total_rating_count"`
	UpdatedAt             int    `json:"updated_at"`
	VersionParent         int    `json:"version_parent"`
	VersionTitle          string `json:"version_title"`
	Websites              []int  `json:"websites"`
	*/
}

type Website struct { // NOTE: more fields available
	Category int    `json:"category"`
	Url      string `json:"url"`
}

type UnixTime time.Time

func (t UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t *UnixTime) UnmarshalJSON(s []byte) (err error) {
	q, err := strconv.ParseInt(string(s), 10, 64)
	if err != nil {
		return err
	}
	*(*time.Time)(t) = time.Unix(q, 0)
	return nil
}
