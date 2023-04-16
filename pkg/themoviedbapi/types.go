package themoviedbapi

type ExternalIds struct {
	Id     int64  `json:"id"`
	ImdbId string `json:"imdb_id"`
}

const (
	MediaTypeMovie = "movie"
	MediaTypeTv    = "tv"
)

type MultiSearchResult struct { // disgusting tagged union, media_type tells us which fields we can expect to be set
	Id           int64  `json:"id"`
	MediaType    string `json:"media_type"`
	Title        string `json:"title"`          // when movie
	Name         string `json:"name"`           // when TV
	ReleaseDate  string `json:"release_date"`   // when movie, yyyy-mm-dd
	FirstAirDate string `json:"first_air_date"` // when TV, yyyy-mm-dd
}

type Movie struct {
	Id             int64       `json:"id"`
	ExternalIds    ExternalIds `json:"external_ids"`
	Title          string      `json:"title"`
	OriginalTitle  string      `json:"original_title"`
	Overview       string      `json:"overview"`
	RuntimeMinutes int         `json:"runtime"`
	ReleaseDate    string      `json:"release_date"` // yyyy-mm-dd
	RevenueDollars int64       `json:"revenue"`
	BackdropPath   string      `json:"backdrop_path"`
}

type Tv struct {
	Id           int64       `json:"id"`
	Name         string      `json:"name"`
	Overview     string      `json:"overview"`
	BackdropPath string      `json:"backdrop_path"`
	PosterPath   string      `json:"poster_path"`
	Homepage     string      `json:"homepage"`
	ExternalIds  ExternalIds `json:"external_ids"`
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

type ImageSize string

const (
	ImageSizeOriginal           ImageSize = "original"
	ImageSizew138_and_h175_face ImageSize = "w138_and_h175_face"
)
