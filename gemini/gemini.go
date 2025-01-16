package gemini

import (
	"fmt"
	"regexp"
	"strconv"

	"gemini-grc/common"
)

// ParseFirstTwoDigits takes a string and returns the first one or two digits as an int.
// If no valid digits are found, it returns an error.
func ParseFirstTwoDigits(input string) (int, error) {
	// Define the regular expression pattern to match one or two leading digits
	re := regexp.MustCompile(`^(\d{1,2})`)

	// Find the first match in the string
	matches := re.FindStringSubmatch(input)
	if len(matches) == 0 {
		return 0, fmt.Errorf("%w", common.ErrGeminiResponseHeader)
	}

	// Parse the captured match as an integer
	snapshot, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("%w: %w", common.ErrTextParse, err)
	}

	return snapshot, nil
}

// extractRedirectTarget returns the redirection
// URL by parsing the header (or error message)
func extractRedirectTarget(currentURL common.URL, input string) (*common.URL, error) {
	// \d+ - matches one or more digits
	// \s+ - matches one or more whitespace
	// ([^\r]+) - captures everything until it hits a \r (or end of string)
	pattern := `\d+\s+([^\r]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%w: %s", common.ErrGeminiRedirect, input)
	}
	newURL, err := common.DeriveAbsoluteURL(currentURL, matches[1])
	if err != nil {
		return nil, fmt.Errorf("%w: %w: %s", common.ErrGeminiRedirect, err, input)
	}
	return newURL, nil
}
