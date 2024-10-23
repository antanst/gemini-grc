package gemini

import (
	"bufio"
	"fmt"
	"strings"
)

// ParseRobotsTxt takes robots.txt content and a host, returns list of full URLs that shouldn't be visited
func ParseRobotsTxt(content string, host string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var disallowedPaths []string

	// Skip everything until we find "User-agent: *" line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.ToLower(line) == "user-agent: *" {
			break
		}
	}

	// Now collect all Disallow paths
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Stop if we hit another User-agent section
		if strings.HasPrefix(strings.ToLower(line), "user-agent:") {
			break
		}

		// Parse Disallow lines
		if strings.HasPrefix(strings.ToLower(line), "disallow:") {
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
