package gemini

import (
	"fmt"
	"net/url"
	"regexp"

	"gemini-grc/common/linkList"
	url2 "gemini-grc/common/url"
	"gemini-grc/logging"
	"gemini-grc/util"
	"github.com/antanst/go_errors"
)

func GetPageLinks(currentURL url2.URL, gemtext string) linkList.LinkList {
	linkLines := util.GetLinesMatchingRegex(gemtext, `(?m)^=>[ \t]+.*`)
	if len(linkLines) == 0 {
		return nil
	}
	var linkURLs linkList.LinkList
	// Normalize URLs in links
	for _, line := range linkLines {
		linkUrl, err := ParseGeminiLinkLine(line, currentURL.String())
		if err != nil {
			logging.LogDebug("error parsing gemini link line: %s", err)
			continue
		}
		linkURLs = append(linkURLs, *linkUrl)
	}
	return linkURLs
}

// ParseGeminiLinkLine takes a single link line and the current URL,
// return the URL converted to an absolute URL
// and its description.
func ParseGeminiLinkLine(linkLine string, currentURL string) (*url2.URL, error) {
	// Check: currentURL is parseable
	baseURL, err := url.Parse(currentURL)
	if err != nil {
		return nil, go_errors.NewError(fmt.Errorf("error parsing link line: %w input '%s'", err, linkLine))
	}

	// Extract the actual URL and the description
	re := regexp.MustCompile(`^=>[ \t]+(\S+)([ \t]+.*)?`)
	matches := re.FindStringSubmatch(linkLine)
	if len(matches) == 0 {
		return nil, go_errors.NewError(fmt.Errorf("error parsing link line: no regexp match for line %s", linkLine))
	}

	originalURLStr := matches[1]

	// Check: Unescape the URL if escaped
	_, err = url.QueryUnescape(originalURLStr)
	if err != nil {
		return nil, go_errors.NewError(fmt.Errorf("error parsing link line: %w input '%s'", err, linkLine))
	}

	description := ""
	if len(matches) > 2 {
		description = matches[2]
	}

	// Parse the URL from the link line
	parsedURL, err := url.Parse(originalURLStr)
	if err != nil {
		return nil, go_errors.NewError(fmt.Errorf("error parsing link line: %w input '%s'", err, linkLine))
	}

	// If link URL is relative, resolve full URL
	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// remove usual first space from URL description:
	// => URL description
	//       ^^^^^^^^^^^^
	if len(description) > 0 && description[0] == ' ' {
		description = description[1:]
	}

	finalURL, err := url2.ParseURL(parsedURL.String(), description, true)
	if err != nil {
		return nil, go_errors.NewError(fmt.Errorf("error parsing link line: %w input '%s'", err, linkLine))
	}

	return finalURL, nil
}
