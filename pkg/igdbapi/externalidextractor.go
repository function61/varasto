package igdbapi

import (
	"fmt"
	"regexp"
)

type ExternalIds struct {
	Official             *string // official website, "homepage"
	SteamId              *string
	GogSlug              *string
	EnglishWikipediaSlug *string
	RedditSlug           *string
	GooglePlayAppId      *string
	AppleAppStoreAppId   *string
}

type idExtractor func(url string, ids *ExternalIds) error

// keyed by IGDB's website "category" (= specific website)
var extractorByCategory = map[int]idExtractor{
	// <anything>
	WebsiteOfficial: func(url string, ids *ExternalIds) error {
		ids.Official = &url
		return nil
	},
	// https://store.steampowered.com/app/270910
	WebsiteSteam: regex(`steampowered.com/app/(\d+)`, func(key string, ids *ExternalIds) { ids.SteamId = &key }),
	// https://www.gog.com/game/worms_world_party_remastered
	WebsiteGog: regex(`gog.com/game/([^/]+)`, func(key string, ids *ExternalIds) { ids.GogSlug = &key }),
	// https://en.wikipedia.org/wiki/Battle_City_(video_game)
	WebsiteWikipedia: regex(`en.wikipedia.org/wiki/(.+)`, func(key string, ids *ExternalIds) { ids.EnglishWikipediaSlug = &key }),
	// https://www.reddit.com/r/dukenukem/
	WebsiteReddit: regex(`reddit.com/r/([^/]+)`, func(key string, ids *ExternalIds) { ids.RedditSlug = &key }),
	// https://play.google.com/store/apps/details?id=com.frogmind.badland&hl=en
	WebsiteAndroid: regex(`\?id=([^&]+)`, func(key string, ids *ExternalIds) { ids.GooglePlayAppId = &key }),
	// https://itunes.apple.com/us/app/badland/id535176909?mt=8&uo=4
	WebsiteIphone: regex(`/id(\d+)`, func(key string, ids *ExternalIds) { ids.AppleAppStoreAppId = &key }),
	WebsiteIpad:   regex(`/id(\d+)`, func(key string, ids *ExternalIds) { ids.AppleAppStoreAppId = &key }),
}

func regex(expression string, assign func(string, *ExternalIds)) idExtractor {
	idRegex := regexp.MustCompile(expression)

	return func(url string, ids *ExternalIds) error {
		matches := idRegex.FindStringSubmatch(url)
		if matches == nil {
			return fmt.Errorf("unable to parse ID from URL: %s", url)
		}

		assign(matches[1], ids)

		return nil
	}
}
