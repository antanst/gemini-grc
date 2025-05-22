package gemini

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	commonErrors "gemini-grc/common/errors"
	"gemini-grc/common/snapshot"
	"github.com/guregu/null/v5"
)

// ProcessData processes the raw data from a Gemini response and populates the Snapshot.
// This function is exported for use by the robotsMatch package.
func ProcessData(s snapshot.Snapshot, data []byte) (*snapshot.Snapshot, error) {
	header, body, err := getHeadersAndData(data)
	if err != nil {
		return &s, err
	}
	code, mimeType, lang := getMimeTypeAndLang(header)

	if code != 0 {
		s.ResponseCode = null.IntFrom(int64(code))
	}
	if header != "" {
		s.Header = null.StringFrom(header)
	}
	if mimeType != "" {
		s.MimeType = null.StringFrom(mimeType)
	}
	if lang != "" {
		s.Lang = null.StringFrom(lang)
	}

	// If we've got a Gemini document, populate
	// `GemText` field, otherwise raw data goes to `Data`.
	if mimeType == "text/gemini" {
		validBody, err := BytesToValidUTF8(body)
		if err != nil {
			return nil, err
		}
		s.GemText = null.StringFrom(validBody)
	} else {
		s.Data = null.ValueFrom(body)
	}
	return &s, nil
}

// Checks for a Gemini header, which is
// basically the first line of the response
// and should contain the response code,
// mimeType and language.
func getHeadersAndData(data []byte) (string, []byte, error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, commonErrors.NewHostError(fmt.Errorf("error parsing header"))
	}
	firstLine := string(data[:firstLineEnds])
	rest := data[firstLineEnds+1:]
	return strings.TrimSpace(firstLine), rest, nil
}

// getMimeTypeAndLang Parses code, mime type and language
// given a Gemini header.
func getMimeTypeAndLang(headers string) (int, string, string) {
	// First try to match the full format: "<code> <mimetype> [charset=<value>] [lang=<value>]"
	// The regex looks for:
	// - A number (\d+)
	// - Followed by whitespace and a mimetype ([a-zA-Z0-9/\-+]+)
	// - Optionally followed by charset and/or lang parameters in any order
	// - Only capturing the lang value, ignoring charset
	re := regexp.MustCompile(`^(\d+)\s+([a-zA-Z0-9/\-+]+)(?:(?:[\s;]+(?:charset=[^;\s]+|lang=([a-zA-Z0-9-]+)))*)\s*$`)
	matches := re.FindStringSubmatch(headers)
	if len(matches) <= 1 {
		// If full format doesn't match, try to match redirect format: "<code> <URL>"
		// This handles cases like "31 gemini://example.com"
		re := regexp.MustCompile(`^(\d+)\s+(.+)$`)
		matches := re.FindStringSubmatch(headers)
		if len(matches) <= 1 {
			// If redirect format doesn't match, try to match just a status code
			// This handles cases like "99"
			re := regexp.MustCompile(`^(\d+)\s*$`)
			matches := re.FindStringSubmatch(headers)
			if len(matches) <= 1 {
				return 0, "", ""
			}
			code, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, "", ""
			}
			return code, "", ""
		}
		code, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, "", ""
		}
		return code, "", ""
	}
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", ""
	}
	mimeType := matches[2]
	lang := matches[3] // Will be empty string if no lang parameter was found
	return code, mimeType, lang
}

func isGeminiCapsule(s *snapshot.Snapshot) bool {
	return !s.Error.Valid && s.MimeType.Valid && s.MimeType.String == "text/gemini"
}
