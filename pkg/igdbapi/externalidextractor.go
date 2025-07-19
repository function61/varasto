package igdbapi

import (
	"fmt"
	"regexp"
)

type ExternalIDs struct {
	Official             *string // official website, "homepage"
	SteamID              *string
	GogSlug              *string
	EnglishWikipediaSlug *string
	RedditSlug           *string
	GooglePlayAppID      *string
	AppleAppStoreAppID   *string
}

type idExtractor func(url string, ids *ExternalIDs) error

// keyed by IGDB's website "category" (= specific website)
var extractorByCategory = map[int]idExtractor{
	// <anything>
	WebsiteOfficial: func(url string, ids *ExternalIDs) error {
		ids.Official = &url
		return nil
	},
	// https://store.steampowered.com/app/270910
	WebsiteSteam: regex(`steampowered.com/app/(\d+)`, func(key string, ids *ExternalIDs) { ids.SteamID = &key }),
	// https://www.gog.com/game/worms_world_party_remastered
	WebsiteGog: regex(`gog.com/game/([^/]+)`, func(key string, ids *ExternalIDs) { ids.GogSlug = &key }),
	// https://en.wikipedia.org/wiki/Battle_City_(video_game)
	// https://wikipedia.org/wiki/Sonic_Colors
	WebsiteWikipedia: regex(`(?:en.)?wikipedia.org/wiki/(.+)`, func(key string, ids *ExternalIDs) { ids.EnglishWikipediaSlug = &key }),
	// https://www.reddit.com/r/dukenukem/
	WebsiteReddit: regex(`reddit.com/r/([^/]+)`, func(key string, ids *ExternalIDs) { ids.RedditSlug = &key }),
	// https://play.google.com/store/apps/details?id=com.frogmind.badland&hl=en
	WebsiteAndroid: regex(`\?id=([^&]+)`, func(key string, ids *ExternalIDs) { ids.GooglePlayAppID = &key }),
	// https://itunes.apple.com/us/app/badland/id535176909?mt=8&uo=4
	WebsiteIphone: regex(`/id(\d+)`, func(key string, ids *ExternalIDs) { ids.AppleAppStoreAppID = &key }),
	WebsiteIpad:   regex(`/id(\d+)`, func(key string, ids *ExternalIDs) { ids.AppleAppStoreAppID = &key }),
}

func regex(expression string, assign func(string, *ExternalIDs)) idExtractor {
	idRegex := regexp.MustCompile(expression)

	return func(url string, ids *ExternalIDs) error {
		matches := idRegex.FindStringSubmatch(url)
		if matches == nil {
			return fmt.Errorf("unable to parse ID from URL: %s", url)
		}

		assign(matches[1], ids)

		return nil
	}
}
