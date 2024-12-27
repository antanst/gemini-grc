package gemini

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"gemini-grc/logging"
)

func GetPageLinks(currentURL URL, gemtext string) LinkList {
	// Grab link lines
	linkLines := ExtractLinkLines(gemtext)
	if len(linkLines) == 0 {
		return nil
	}
	var linkURLs LinkList
	// Normalize URLs in links, and store them in snapshot
	for _, line := range linkLines {
		linkURL, err := NormalizeLink(line, currentURL.String())
		if err != nil {
			logging.LogDebug("%s: %s", ErrGeminiLinkLineParse, err)
			continue
		}
		linkURLs = append(linkURLs, *linkURL)
	}
	return linkURLs
}

// ExtractLinkLines takes a Gemtext document as a string and returns all lines that are link lines
func ExtractLinkLines(gemtext string) []string {
	// Define the regular expression pattern to match link lines
	re := regexp.MustCompile(`(?m)^=>[ \t]+.*`)

	// Find all matches using the regular expression
	matches := re.FindAllString(gemtext, -1)

	return matches
}

// NormalizeLink takes a single link line and the current URL,
// return the URL converted to an absolute URL
// and its description.
func NormalizeLink(linkLine string, currentURL string) (*URL, error) {
	// Parse the current URL
	baseURL, err := url.Parse(currentURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrURLParse, err)
	}

	// Regular expression to extract the URL part from a link line
	re := regexp.MustCompile(`^=>[ \t]+(\S+)([ \t]+.*)?`)

	// Use regex to extract the URL and the rest of the line
	matches := re.FindStringSubmatch(linkLine)
	if len(matches) == 0 {
		// If the line doesn't match the expected format, return it unchanged
		return nil, fmt.Errorf("%w for link line %s", ErrGeminiLinkLineParse, linkLine)
	}

	originalURLStr := matches[1]
	_, err = url.QueryUnescape(originalURLStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrURLDecode, err)
	}

	restOfLine := ""
	if len(matches) > 2 {
		restOfLine = matches[2]
	}

	// Parse the URL from the link line
	parsedURL, err := url.Parse(originalURLStr)
	if err != nil {
		// If URL parsing fails, return an error
		return nil, fmt.Errorf("%w: %w", ErrURLParse, err)
	}

	// Resolve relative URLs against the base URL
	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Remove usual first space from URL description:
	// => URL description
	//       ^^^^^^^^^^^^
	if len(restOfLine) > 0 && restOfLine[0] == ' ' {
		restOfLine = restOfLine[1:]
	}

	finalURL, err := ParseURL(parsedURL.String(), restOfLine)
	if err != nil {
		// If URL parsing fails, return an error
		return nil, fmt.Errorf("%w: %w", ErrURLParse, err)
	}

	return finalURL, nil
}

// ParseFirstTwoDigits takes a string and returns the first one or two digits as an int.
// If no valid digits are found, it returns an error.
func ParseFirstTwoDigits(input string) (int, error) {
	// Define the regular expression pattern to match one or two leading digits
	re := regexp.MustCompile(`^(\d{1,2})`)

	// Find the first match in the string
	matches := re.FindStringSubmatch(input)
	if len(matches) == 0 {
		return 0, fmt.Errorf("%w", ErrGeminiResponseHeader)
	}

	// Parse the captured match as an integer
	snapshot, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrTextParse, err)
	}

	return snapshot, nil
}

// extractRedirectTarget returns the redirection
// URL by parsing the header (or error message)
func extractRedirectTarget(currentURL URL, input string) (*URL, error) {
	// \d+ - matches one or more digits
	// \s+ - matches one or more whitespace
	// ([^\r]+) - captures everything until it hits a \r (or end of string)
	pattern := `\d+\s+([^\r]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%w: %s", ErrGeminiRedirect, input)
	}
	newURL, err := DeriveAbsoluteURL(currentURL, matches[1])
	if err != nil {
		return nil, fmt.Errorf("%w: %w: %s", ErrGeminiRedirect, err, input)
	}
	return newURL, nil
}
