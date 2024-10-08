package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func checkGeminiStatusCode(code int) error {
	switch {
	case code == 20:
		return nil
	case code >= 10 && code < 20:
		return fmt.Errorf("Gemini response %d needs data input", code)
	case code >= 30 && code < 40:
		return fmt.Errorf("Gemini response %d redirect", code)
	case code >= 40 && code < 50:
		return fmt.Errorf("Gemini response %d server error", code)
	case code >= 50 && code < 60:
		return fmt.Errorf("Gemini response %d server permanent error", code)
	case code >= 60 && code < 70:
		return fmt.Errorf("Gemini response %d certificate error", code)
	default:
		return fmt.Errorf("Unexpected/unhandled Gemini response %d", code)
	}
}

func parseHeaders(data string) (string, string) {
	re := regexp.MustCompile(`^\d+\s+([a-zA-Z0-9/\-+]+)[;\s]+(lang=([a-zA-Z0-9-]+))?`)
	matches := re.FindStringSubmatch(data)
	if matches == nil || len(matches) <= 1 {
		return "", ""
	}
	return matches[1], matches[3]
}

func ProcessHeaders(snapshot *Snapshot) *Snapshot {
	LogDebug("[%s] Processing snapshot", snapshot.URL.String())
	mimetype, lang := parseHeaders(snapshot.Data)
	if mimetype != "" {
		snapshot.MimeType = mimetype
	}
	if lang != "" {
		snapshot.Lang = lang
	}
	return snapshot
}

func ProcessGemini(snapshot *Snapshot) *Snapshot {
	code, err := ParseFirstTwoDigits(snapshot.Data)
	if err != nil {
		snapshot.Error = fmt.Errorf("[%s] No/invalid gemini response code", snapshot.URL.String())
		return snapshot
	}
	snapshot.ResponseCode = code

	// Remove response headers from body (first line)
	index := strings.Index(snapshot.Data, "\n")
	if index != -1 {
		snapshot.Data = snapshot.Data[index+1:]
	}

	// Grab any link lines
	linkLines := ExtractLinkLines(snapshot.Data)
	LogDebug("[%s] Found %d links", snapshot.URL.String(), len(linkLines))

	// Normalize URLs in links, and store them in snapshot
	for _, line := range linkLines {
		normalizedLink, descr, error := NormalizeLink(line, snapshot.URL.String())
		if error != nil {
			LogError("[%s] Invalid link URL %w", snapshot.URL.String(), error)
			continue
		}
		geminiUrl, error := ParseUrl(normalizedLink, descr)
		if error != nil {
			LogError("[%s] Unparseable gemini link %w", snapshot.URL.String(), error)
		}
		snapshot.Links = append(snapshot.Links, *geminiUrl)
	}
	return snapshot
}

func SaveResult(rootPath string, s *Snapshot) {
	parentPath := path.Join(rootPath, s.URL.Hostname)
	urlPath := s.URL.Path
	// If path is empty, add `index.gmi` as the file to save
	if urlPath == "" || urlPath == "." {
		urlPath = fmt.Sprintf("index.gmi")
	}
	// If path ends with '/' then add index.gmi for the
	// directory to be created.
	if strings.HasSuffix(urlPath, "/") {
		urlPath = strings.Join([]string{urlPath, "index.gmi"}, "")
	}

	finalPath, err := calcFilePath(parentPath, urlPath)
	if err != nil {
		LogError("Error saving %s: %w", s.URL, err)
		return
	}
	// Ensure the directory exists
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		LogError("Failed to create directory: %w", err)
		return
	}
	err = os.WriteFile(finalPath, []byte((*s).Data), 0666)
	if err != nil {
		LogError("Error saving %s: %w", s.URL.Full, err)
	}
}

func ParseUrl(input string, descr string) (*GeminiUrl, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %s: %w", input, err)
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	str_port := u.Port()
	path := u.Path
	if str_port == "" {
		str_port = "1965"
	}
	port, err := strconv.Atoi(str_port)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %s: %w", input, err)
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

// Take a single link line and the current URL,
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
		return "", "", fmt.Errorf("Not a link line: %v", linkLine)
	}

	originalURLStr := matches[1]
	restOfLine := ""
	if len(matches) > 2 {
		restOfLine = matches[2]
	}

	// Parse the URL from the link line
	parsedURL, err := url.Parse(originalURLStr)
	if err != nil {
		// If URL parsing fails, return an error
		return "", "", fmt.Errorf("Invalid URL in link line '%s': %v", originalURLStr, err)
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
