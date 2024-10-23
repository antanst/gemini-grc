package gemini

import (
	"errors"
	"fmt"
	"gemini-grc/logging"
	"net/url"
	gourl "net/url"
	"regexp"
	"strconv"
	"strings"
)

func isGeminiURL(url string) bool {
	_, err := gourl.Parse(url)
	if err != nil {
		logging.LogWarn("[%s] Invalid URL: %v", url, err)
		return false
	}
	return strings.HasPrefix(url, "gemini://")
}

func parseLinks(s Snapshot, queue chan string) {
	for _, link := range *s.Links {
		if strings.HasPrefix(link.Full, "gemini://") {
			go func(link GeminiUrl) {
				// fmt.Printf("LINK: %s\n", link)
				queue <- link.Full
			}(link)
		}
	}
}

func checkGeminiStatusCode(code int) error {
	switch {
	case code == 20:
		return nil
	case code >= 10 && code < 20:
		return fmt.Errorf("gemini response %d needs data input", code)
	case code >= 30 && code < 40:
		return fmt.Errorf("gemini response %d redirect", code)
	case code >= 40 && code < 50:
		return fmt.Errorf("gemini response %d server error", code)
	case code >= 50 && code < 60:
		return fmt.Errorf("gemini response %d server permanent error", code)
	case code >= 60 && code < 70:
		return fmt.Errorf("gemini response %d certificate error", code)
	default:
		return fmt.Errorf("unexpected/unhandled Gemini response %d", code)
	}
}

func ProcessGemini(snapshot *Snapshot) *Snapshot {
	// Grab link lines
	linkLines := ExtractLinkLines(snapshot.GemText.String)
	logging.LogDebug("[%s] Found %d links", snapshot.URL.String(), len(linkLines))

	// Normalize URLs in links, and store them in snapshot
	for _, line := range linkLines {
		normalizedLink, descr, err := NormalizeLink(line, snapshot.URL.String())
		if err != nil {
			logging.LogDebug("Cannot normalize URL in line '%s': %v", line, err)
			continue
		}
		geminiUrl, err := ParseUrl(normalizedLink, descr)
		if err != nil {
			logging.LogDebug("Cannot parse URL in link '%s': %v", line, err)
			continue
		}
		if snapshot.Links == nil {
			snapshot.Links = &LinkList{*geminiUrl}
		} else {
			*snapshot.Links = append(*snapshot.Links, *geminiUrl)
		}
	}
	return snapshot
}

func ParseUrl(input string, descr string) (*GeminiUrl, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL %s: %w", input, err)
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	strPort := u.Port()
	path := u.Path
	if strPort == "" {
		strPort = "1965"
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL %s: %w", input, err)
	}
	return &GeminiUrl{Protocol: protocol, Hostname: hostname, Port: port, Path: path, Descr: descr, Full: u.String()}, nil
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
func NormalizeLink(linkLine string, currentURL string) (link string, descr string, err error) {
	// Parse the current URL
	baseURL, err := url.Parse(currentURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid current URL: %v", err)
	}

	// Regular expression to extract the URL part from a link line
	re := regexp.MustCompile(`^=>[ \t]+(\S+)([ \t]+.*)?`)

	// Use regex to extract the URL and the rest of the line
	matches := re.FindStringSubmatch(linkLine)
	if len(matches) == 0 {
		// If the line doesn't match the expected format, return it unchanged
		return "", "", fmt.Errorf("not a link line: %v", linkLine)
	}

	originalURLStr := matches[1]
	_, err = url.QueryUnescape(originalURLStr)
	if err != nil {
		return "", "", fmt.Errorf("error decoding URL: %w", err)
	}

	restOfLine := ""
	if len(matches) > 2 {
		restOfLine = matches[2]
	}

	// Parse the URL from the link line
	parsedURL, err := url.Parse(originalURLStr)
	if err != nil {
		// If URL parsing fails, return an error
		return "", "", fmt.Errorf("invalid URL '%s': %v", originalURLStr, err)
	}

	// Resolve relative URLs against the base URL
	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Construct the canonicalized link line
	canonicalURLStr := parsedURL.String()

	// Remove usual first space from URL description:
	// => URL description
	//       ^^^^^^^^^^^^
	if len(restOfLine) > 0 && restOfLine[0] == ' ' {
		restOfLine = restOfLine[1:]
	}

	return canonicalURLStr, restOfLine, nil
	//	canonicalizedLine := fmt.Sprintf("=> %s%s", canonicalURLStr, restOfLine)
	// return canonicalizedLine, nil
}

// ParseFirstTwoDigits takes a string and returns the first one or two digits as an int.
// If no valid digits are found, it returns an error.
func ParseFirstTwoDigits(input string) (int, error) {
	// Define the regular expression pattern to match one or two leading digits
	re := regexp.MustCompile(`^(\d{1,2})`)

	// Find the first match in the string
	matches := re.FindStringSubmatch(input)
	if len(matches) == 0 {
		return 0, errors.New("no digits found at the beginning of the string")
	}

	// Parse the captured match as an integer
	snapshot, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to convert matched digits to int: %v", err)
	}

	return snapshot, nil
}
