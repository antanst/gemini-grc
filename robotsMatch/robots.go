package robotsMatch

import (
	"context"
	"fmt"
	"strings"

	"gemini-grc/common/contextlog"
	"gemini-grc/contextutil"
	"git.antanst.com/antanst/logging"
)

// ParseRobotsTxt takes robots.txt content and a host, and
// returns a list of full URLs that shouldn't be visited.
// This is the legacy version without context support.
// TODO Also take into account the user agent?
// Check gemini://geminiprotocol.net/docs/companion/robots.gmi
func ParseRobotsTxt(content string, host string) []string {
	// Call the context-aware version with a background context
	return ParseRobotsTxtWithContext(context.Background(), content, host)
}

// ParseRobotsTxtWithContext takes robots.txt content and a host, and
// returns a list of full URLs that shouldn't be visited.
// This version supports context for logging.
// TODO Also take into account the user agent?
// Check gemini://geminiprotocol.net/docs/companion/robots.gmi
func ParseRobotsTxtWithContext(ctx context.Context, content string, host string) []string {
	// Create a context for robots.txt parsing
	parseCtx := contextutil.ContextWithComponent(ctx, "robotsMatch.parser")

	var disallowedPaths []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "disallow:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[1])
				if path != "" {
					// Construct full Gemini URL
					var fullURL string

					// Handle if the path is already a full URL
					if strings.HasPrefix(path, "gemini://") {
						// Extract just the path from the full URL
						urlParts := strings.SplitN(path, "/", 4)
						if len(urlParts) >= 4 {
							// Get the path part (everything after the domain)
							pathPart := "/" + urlParts[3]
							fullURL = fmt.Sprintf("gemini://%s%s", host, pathPart)
						} else {
							// If it's just a domain without a path, skip it or use root path
							fullURL = fmt.Sprintf("gemini://%s/", host)
						}
					} else {
						// It's a relative path, just add it to the host
						if !strings.HasPrefix(path, "/") {
							path = "/" + path
						}
						fullURL = fmt.Sprintf("gemini://%s%s", host, path)
					}

					disallowedPaths = append(disallowedPaths, fullURL)

					// Add additional logging to debug robots.txt parsing
					contextlog.LogDebugWithContext(parseCtx, logging.GetSlogger(), "Added robots.txt disallow rule: %s from original: %s", fullURL, path)
				}
			}
		}
	}
	return disallowedPaths
}
