package robotsMatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"gemini-grc/common/contextlog"
	"gemini-grc/common/snapshot"
	geminiUrl "gemini-grc/common/url"
	"gemini-grc/contextutil"
	"gemini-grc/gemini"
	"gemini-grc/logging"
)

// RobotsCache is a map of blocked URLs
// key: URL
// value: []string list of disallowed URLs
// If a key has no blocked URLs, an empty
// list is stored for caching.
var RobotsCache sync.Map //nolint:gochecknoglobals

func populateRobotsCache(ctx context.Context, key string) (entries []string, _err error) {
	// Create a context for robots cache population
	cacheCtx := contextutil.ContextWithComponent(ctx, "robotsCache")

	// We either store an empty list when
	// no rules, or a list of disallowed URLs.
	// This applies even if we have an error
	// finding/downloading robots.txt
	defer func() {
		RobotsCache.Store(key, entries)
	}()

	url := fmt.Sprintf("gemini://%s/robots.txt", key)
	contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "Fetching robots.txt from %s", url)

	// Use the context-aware version to honor timeout and cancellation
	robotsContent, err := gemini.ConnectAndGetDataWithContext(cacheCtx, url)
	if err != nil {
		// Check for context timeout or cancellation specifically
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "Timeout or cancellation while fetching robots.txt: %v", err)
			// Don't cache the result on timeout, to allow retrying later
			return []string{}, err
		}
		// For other errors, we store an empty list for this host
		// to avoid continually hitting it
		contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "Failed to get robots.txt: %v", err)
		RobotsCache.Store(key, []string{})
		return []string{}, err
	}

	s, err := snapshot.SnapshotFromURL(url, true)
	if err != nil {
		contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "Failed to create snapshot from URL: %v", err)
		return []string{}, nil
	}

	// TODO: Update gemini.ProcessData to accept context
	s, err = gemini.ProcessData(*s, robotsContent)
	if err != nil {
		contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "robots.txt error: %s", err)
		return []string{}, nil
	}

	if s.ResponseCode.ValueOrZero() != 20 {
		contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "robots.txt error code %d, ignoring", s.ResponseCode.ValueOrZero())
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
		contextlog.LogDebugWithContext(cacheCtx, logging.GetSlogger(), "Unsupported mime type: %s", s.MimeType.ValueOrZero())
		return []string{}, nil
	}

	entries = ParseRobotsTxtWithContext(ctx, data, key)
	return entries, nil
}

// RobotMatch checks if the snapshot URL matches
// a robots.txt allow rule.
func RobotMatch(ctx context.Context, u string) (bool, error) {
	// Create a context for robots operations
	robotsCtx := contextutil.ContextWithComponent(ctx, "robotsMatch")

	url, err := geminiUrl.ParseURL(u, "", true)
	if err != nil {
		contextlog.LogErrorWithContext(robotsCtx, logging.GetSlogger(), "Failed to parse URL: %v", err)
		return false, err
	}

	key := strings.ToLower(fmt.Sprintf("%s:%d", url.Hostname, url.Port))
	contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "Checking robots.txt for URL: %s with host key: %s", u, key)

	var disallowedURLs []string
	cacheEntries, ok := RobotsCache.Load(key)
	if !ok {
		// First time check, populate robot cache
		contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "No robots.txt cache for %s, fetching...", key)
		var fetchErr error
		disallowedURLs, fetchErr = populateRobotsCache(ctx, key)
		if fetchErr != nil {
			contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "Error populating robots.txt cache for %s: %v", key, fetchErr)

			// Handle context timeouts by propagating the error
			if errors.Is(fetchErr, context.DeadlineExceeded) || errors.Is(fetchErr, context.Canceled) {
				contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "Timeout or cancellation while checking robots.txt")
				return false, fetchErr
			}

			// For other errors, assume we can proceed without robots.txt
			return false, nil
		}
		if len(disallowedURLs) > 0 {
			contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "Added to robots.txt cache: %v => %v", key, disallowedURLs)
		} else {
			contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "No disallowed paths found in robots.txt for %s", key)
		}
	} else {
		var ok bool
		disallowedURLs, ok = cacheEntries.([]string)
		if !ok {
			contextlog.LogErrorWithContext(robotsCtx, logging.GetSlogger(), "Invalid type in robots.txt cache for %s", key)
			disallowedURLs = []string{} // Use empty list as fallback
		}
		contextlog.LogDebugWithContext(robotsCtx, logging.GetSlogger(), "Found %d disallowed paths in robots.txt cache for %s", len(disallowedURLs), key)
	}
	return isURLblocked(ctx, disallowedURLs, url.Full), nil
}

// Initialize initializes the robots.txt match package
func Initialize() error {
	logging.LogDebug("Initializing robotsMatch package")
	return nil
}

// Shutdown cleans up the robots.txt match package
func Shutdown() error {
	logging.LogDebug("Shutting down robotsMatch package")
	return nil
}

func isURLblocked(ctx context.Context, disallowedURLs []string, input string) bool {
	// Create a context for URL blocking checks
	blockCtx := contextutil.ContextWithComponent(ctx, "robotsMatch.isURLblocked")

	inputLower := strings.ToLower(input)
	contextlog.LogDebugWithContext(blockCtx, logging.GetSlogger(), "Checking URL against robots.txt rules: %s", input)

	for _, url := range disallowedURLs {
		urlLower := strings.ToLower(url)
		contextlog.LogDebugWithContext(blockCtx, logging.GetSlogger(), "Comparing against rule: %s (lower: %s vs %s)", url, inputLower, urlLower)

		if strings.HasPrefix(inputLower, urlLower) {
			contextlog.LogDebugWithContext(blockCtx, logging.GetSlogger(), "MATCH! robots.txt rule: %s blocks URL: %s", url, input)
			return true
		}
	}
	contextlog.LogDebugWithContext(blockCtx, logging.GetSlogger(), "No robots.txt rules matched URL: %s", input)
	return false
}
