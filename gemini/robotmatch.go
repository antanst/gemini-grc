package gemini

import (
	"fmt"
	"strings"
	"sync"

	"gemini-grc/common/snapshot"
	geminiUrl "gemini-grc/common/url"
	"gemini-grc/logging"
)

// RobotsCache is a map of blocked URLs
// key: URL
// value: []string list of disallowed URLs
// If a key has no blocked URLs, an empty
// list is stored for caching.
var RobotsCache sync.Map //nolint:gochecknoglobals

func populateRobotsCache(key string) (entries []string, _err error) {
	// We either store an empty list when
	// no rules, or a list of disallowed URLs.
	// This applies even if we have an error
	// finding/downloading robots.txt
	defer func() {
		RobotsCache.Store(key, entries)
	}()
	url := fmt.Sprintf("gemini://%s/robots.txt", key)
	robotsContent, err := ConnectAndGetData(url)
	if err != nil {
		return []string{}, err
	}
	s, err := snapshot.SnapshotFromURL(url, true)
	if err != nil {
		return []string{}, nil
	}
	s, err = processData(*s, robotsContent)
	if err != nil {
		logging.LogDebug("robots.txt error %s", err)
		return []string{}, nil
	}
	if s.ResponseCode.ValueOrZero() != 20 {
		logging.LogDebug("robots.txt error code %d, ignoring", s.ResponseCode.ValueOrZero())
		return []string{}, nil
	}
	// Some return text/plain, others text/gemini.
	// According to spec, the first is correct,
	// however let's be lenient
	var data string
	switch {
	case s.MimeType.ValueOrZero() == "text/plain":
		data = string(s.Data.ValueOrZero())
	case s.MimeType.ValueOrZero() == "text/gemini":
		data = s.GemText.ValueOrZero()
	default:
		return []string{}, nil
	}
	entries = ParseRobotsTxt(data, key)
	return entries, nil
}

// RobotMatch checks if the snapshot URL matches
// a robots.txt allow rule.
func RobotMatch(u string) (bool, error) {
	url, err := geminiUrl.ParseURL(u, "", true)
	if err != nil {
		return false, err
	}
	key := strings.ToLower(fmt.Sprintf("%s:%d", url.Hostname, url.Port))
	var disallowedURLs []string
	cacheEntries, ok := RobotsCache.Load(key)
	if !ok {
		// First time check, populate robot cache
		disallowedURLs, err := populateRobotsCache(key)
		if err != nil {
			return false, err
		}
		if len(disallowedURLs) > 0 {
			logging.LogDebug("Added to robots.txt cache: %v => %v", key, disallowedURLs)
		}
	} else {
		disallowedURLs, _ = cacheEntries.([]string)
	}
	return isURLblocked(disallowedURLs, url.Full), nil
}

func isURLblocked(disallowedURLs []string, input string) bool {
	for _, url := range disallowedURLs {
		if strings.HasPrefix(strings.ToLower(input), url) {
			logging.LogDebug("robots.txt match: %s matches %s", input, url)
			return true
		}
	}
	return false
}
