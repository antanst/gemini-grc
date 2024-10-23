package gemini

import (
	"fmt"
	"gemini-grc/logging"
	"strings"
	"sync"
)

// key: "host:port" (string)
// value:
// empty []string if no robots data, or
// list of URL prefixes ([]string) in robots
var RobotsCache sync.Map

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
	if robotsData.MimeType == "text/plain" {
		data = string(robotsData.Data)
	} else if robotsData.MimeType == "text/gemini" {
		data = robotsData.GemText
	} else {
		return []string{}
	}
	entries = ParseRobotsTxt(string(data), key)
	return entries
}

// Check if the snapshot URL matches
// a robots.txt allow rule.
func RobotMatch(s *Snapshot) bool {
	logging.LogDebug("Checking robots.txt cache for %s", s.URL.String())
	key := fmt.Sprintf("%s:%d", s.Host, s.URL.Port)
	v, ok := RobotsCache.Load(key)
	if ok == false {
		// First time check, populate robot cache
		logging.LogDebug("No robots.txt entry, populating cache for %s", s.URL.String())
		disallowedURLs := populateBlacklist(key)
		for _, url := range disallowedURLs {
			if strings.HasPrefix(s.URL.String(), url) {
				logging.LogDebug("robots.txt match: %s %s", s.URL.String(), url)
				return true
			}
		}
	} else {
		if len(v.([]string)) == 0 {
			logging.LogDebug("No robots.txt or no rules, allowed")
			return false
		}
		for _, url := range v.([]string) {
			if strings.HasPrefix(s.URL.String(), url) {
				logging.LogDebug("robots.txt match: %s %s", s.URL.String(), url)
				return true
			}
		}
	}
	return false
}
