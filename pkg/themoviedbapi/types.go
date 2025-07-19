package themoviedbapi

import (
	"fmt"
)

type ExternalIDs struct {
	ID     int64  `json:"id"`
	ImdbID string `json:"imdb_id"`
}

const (
	MediaTypeMovie = "movie"
	MediaTypeTv    = "tv"
)

type MultiSearchResult struct { // disgusting tagged union, media_type tells us which fields we can expect to be set
	ID           int64  `json:"id"`
	MediaType    string `json:"media_type"`
	Title        string `json:"title"`          // when movie
	Name         string `json:"name"`           // when TV
	ReleaseDate  string `json:"release_date"`   // when movie, yyyy-mm-dd
	FirstAirDate string `json:"first_air_date"` // when TV, yyyy-mm-dd
}

type Movie struct {
	ID             int64       `json:"id"`
	ExternalIds    ExternalIDs `json:"external_ids"`
	Title          string      `json:"title"`
	OriginalTitle  string      `json:"original_title"`
	Overview       string      `json:"overview"`
	RuntimeMinutes int         `json:"runtime"`
	ReleaseDate    string      `json:"release_date"` // yyyy-mm-dd
	RevenueDollars int64       `json:"revenue"`
	BackdropPath   *ImageID    `json:"backdrop_path"`
}

type Tv struct {
	ID           int64       `json:"id"`
	Name         string      `json:"name"`
	Overview     string      `json:"overview"`
	BackdropPath *ImageID    `json:"backdrop_path"`
	PosterPath   *ImageID    `json:"poster_path"`
	Homepage     string      `json:"homepage"`
	ExternalIds  ExternalIDs `json:"external_ids"`
}

type Episode struct {
	ID            uint64   `json:"id"`
	SeasonNumber  int      `json:"season_number"`
	EpisodeNumber int      `json:"episode_number"`
	Name          string   `json:"name"`
	Overview      string   `json:"overview"`
	AirDate       string   `json:"air_date"` // yyyy-mm-dd
	StillPath     *ImageID `json:"still_path"`
}

type CreditsCastItem struct {
	ID          PersonID
	Name        string
	ProfilePath *ImageID `json:"profile_path"` // actually means profile *picture* path
	Character   string
}

type CreditsCrewItem struct {
	ID          PersonID
	Name        string
	ProfilePath *ImageID `json:"profile_path"` // actually means profile *picture* path
	Job         string
}

type Credits struct {
	Cast []CreditsCastItem
	Crew []CreditsCrewItem
}

func (c *Credits) Director() *string {
	for _, crew := range c.Crew {
		if crew.Job == JobDirector {
			return &crew.Name
		}
	}

	return nil
}

type PersonID int

func (p PersonID) URL() string {
	return fmt.Sprintf("https://www.themoviedb.org/person/%d", p)
}

// something like "/421cSReX2Fktldac8SyY2k0yLwY.jpg" which is used as input to calculate the full image path.
type ImageID string

func (i ImageID) URL(size ImageSize) string {
	return ImagePath(string(i), size)
}

type ImageSize string

const (
	ImageSizeOriginal        ImageSize = "original"
	ImageSizeW138AndH175Face ImageSize = "w138_and_h175_face"
)

type Job string

const (
	JobDirector = "Director"
)
