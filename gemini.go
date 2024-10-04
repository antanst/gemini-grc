package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

func Process(result *Result) *Result {
	LogInfo("[%s] Processing data", result.url.String())
	code, err := ParseFirstTwoDigits(result.data)
	if err != nil {
		result.error = fmt.Errorf("[%s] Invalid gemini response code", result.url.String())
		return result
	}
	if code != 20 {
		result.error = fmt.Errorf("[%s] Gemini response code != 20, skipping", result.url.String())
		return result
	}
	// Grab link lines
	linkLines := ExtractLinkLines(result.data)
	LogDebug("[%s] Found %d links", result.url.String(), len(linkLines))
	// Normalize URLs in links, and store them in result
	for _, line := range linkLines {
		normalizedLink, descr, error := NormalizeLink(line, result.url.String())
		if error != nil {
			LogError("[%s] Invalid link URL %w", result.url.String(), error)
			continue
		}
		geminiUrl, error := ParseUrl(normalizedLink, descr)
		if error != nil {
			LogError("[%s] Unparseable gemini link %w", result.url.String(), error)
		}
		result.links = append(result.links, *geminiUrl)
	}
	return result
}

func ParseUrl(input string, descr string) (*GeminiUrl, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %s: %w", input, err)
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	str_port := u.Port()
	if str_port == "" {
		str_port = "1965"
	}
	port, err := strconv.Atoi(str_port)
	if err != nil {
		return nil, fmt.Errorf("Error parsing URL %s: %w", input, err)
	}
	return &GeminiUrl{protocol: protocol, hostname: hostname, port: port, path: u.Path, descr: descr}, nil
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
	result, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to convert matched digits to int: %v", err)
	}

	return result, nil
}
