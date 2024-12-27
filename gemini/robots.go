package gemini

import (
	"fmt"
	"strings"
)

// ParseRobotsTxt takes robots.txt content and a host, and
// returns a list of full URLs that shouldn't
// be visited.
// TODO Also take into account the user agent?
// Check gemini://geminiprotocol.net/docs/companion/robots.gmi
func ParseRobotsTxt(content string, host string) []string {
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
					disallowedPaths = append(disallowedPaths,
						fmt.Sprintf("gemini://%s%s", host, path))
				}
			}
		}
	}
	return disallowedPaths
}
