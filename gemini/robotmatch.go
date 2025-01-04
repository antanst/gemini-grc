package gemini

import (
	"fmt"
	"gemini-grc/common"
	"strings"
	"sync"

	"gemini-grc/logging"
)

// RobotsCache is a map of blocked URLs
// key: URL
// value: []string list of disallowed URLs
// If a key has no blocked URLs, an empty
// list is stored for caching.
var RobotsCache sync.Map //nolint:gochecknoglobals

func populateBlacklist(key string) (entries []string) {
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
		logging.LogDebug("robots.txt error %s", err)
		return []string{}
	}
	robotsData, err := processData(robotsContent)
	if err != nil {
		logging.LogDebug("robots.txt error %s", err)
		return []string{}
	}
	if robotsData.ResponseCode != 20 {
		logging.LogDebug("robots.txt error code %d, ignoring", robotsData.ResponseCode)
		return []string{}
	}
	// Some return text/plain, others text/gemini.
	// According to spec, the first is correct,
	// however let's be lenient
	var data string
	switch {
	case robotsData.MimeType == "text/plain":
		data = string(robotsData.Data)
	case robotsData.MimeType == "text/gemini":
		data = robotsData.GemText
	default:
		return []string{}
	}
	entries = ParseRobotsTxt(data, key)
	return entries
}

// RobotMatch checks if the snapshot URL matches
// a robots.txt allow rule.
func RobotMatch(u string) bool {
	url, err := common.ParseURL(u, "")
	if err != nil {
		return false
	}
	key := strings.ToLower(fmt.Sprintf("%s:%d", url.Hostname, url.Port))
	logging.LogDebug("Checking robots.txt cache for %s", key)
	var disallowedURLs []string
	cacheEntries, ok := RobotsCache.Load(key)
	if !ok {
		// First time check, populate robot cache
		disallowedURLs = populateBlacklist(key)
		logging.LogDebug("Added to robots.txt cache: %v => %v", key, disallowedURLs)
	} else {
		disallowedURLs, _ = cacheEntries.([]string)
	}
	return isURLblocked(disallowedURLs, url.Full)
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
