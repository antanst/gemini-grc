package gemini

import (
	"fmt"
	"net/url"
	"regexp"

	"gemini-grc/common"
	"gemini-grc/logging"
	"gemini-grc/util"
)

func GetPageLinks(currentURL common.URL, gemtext string) common.LinkList {
	linkLines := util.GetLinesMatchingRegex(gemtext, `(?m)^=>[ \t]+.*`)
	if len(linkLines) == 0 {
		return nil
	}
	var linkURLs common.LinkList
	// Normalize URLs in links
	for _, line := range linkLines {
		linkUrl, err := ParseGeminiLinkLine(line, currentURL.String())
		if err != nil {
			logging.LogDebug("%s: %s", common.ErrGeminiLinkLineParse, err)
			continue
		}
		linkURLs = append(linkURLs, *linkUrl)
	}
	return linkURLs
}

// ParseGeminiLinkLine takes a single link line and the current URL,
// return the URL converted to an absolute URL
// and its description.
func ParseGeminiLinkLine(linkLine string, currentURL string) (*common.URL, error) {
	// Check: currentURL is parseable
	baseURL, err := url.Parse(currentURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrURLParse, err)
	}

	// Extract the actual URL and the description
	re := regexp.MustCompile(`^=>[ \t]+(\S+)([ \t]+.*)?`)
	matches := re.FindStringSubmatch(linkLine)
	if len(matches) == 0 {
		// If the line doesn't match the expected format, return it unchanged
		return nil, fmt.Errorf("%w could not parse gemini link %s", common.ErrGeminiLinkLineParse, linkLine)
	}

	originalURLStr := matches[1]

	// Check: Unescape the URL if escaped
	_, err = url.QueryUnescape(originalURLStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrURLDecode, err)
	}

	description := ""
	if len(matches) > 2 {
		description = matches[2]
	}

	// Parse the URL from the link line
	parsedURL, err := url.Parse(originalURLStr)
	if err != nil {
		// If URL parsing fails, return an error
		return nil, fmt.Errorf("%w: %w", common.ErrURLParse, err)
	}

	// If link URL is relative, resolve full URL
	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Remove usual first space from URL description:
	// => URL description
	//       ^^^^^^^^^^^^
	if len(description) > 0 && description[0] == ' ' {
		description = description[1:]
	}
	
	finalURL, err := common.ParseURL(parsedURL.String(), description)
	if err != nil {
		// If URL parsing fails, return an error
		return nil, fmt.Errorf("%w: %w", common.ErrURLParse, err)
	}

	return finalURL, nil
}
