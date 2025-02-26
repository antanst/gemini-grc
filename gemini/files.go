package gemini

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gemini-grc/common/snapshot"
	"gemini-grc/logging"
)

// sanitizePath encodes invalid filesystem characters using URL encoding.
// Example:
// /example/path/to/page?query=param&another=value
// would become
// example/path/to/page%3Fquery%3Dparam%26another%3Dvalue
func sanitizePath(p string) string {
	// Split the path into its components
	components := strings.Split(p, "/")

	// Encode each component separately
	for i, component := range components {
		// Decode any existing percent-encoded characters
		decodedComponent, err := url.PathUnescape(component)
		if err != nil {
			decodedComponent = component // Fallback to original if unescape fails
		}

		// Encode the component to escape invalid filesystem characters
		encodedComponent := url.QueryEscape(decodedComponent)

		// Replace '+' (from QueryEscape) with '%20' to handle spaces correctly
		encodedComponent = strings.ReplaceAll(encodedComponent, "+", "%20")

		components[i] = encodedComponent
	}

	// Rejoin the components into a sanitized path
	safe := filepath.Join(components...)

	return safe
}

// getFilePath constructs a safe file path from the root path and URL path.
// It URL-encodes invalid filesystem characters to ensure the path is valid.
func calcFilePath(rootPath, urlPath string) (string, error) {
	// Normalize the URL path
	cleanPath := filepath.Clean(urlPath)

	// Safe check to prevent directory traversal
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("Invalid URL path: contains directory traversal")
	}

	// Sanitize the path by encoding invalid characters
	safePath := sanitizePath(cleanPath)

	// Join the root path and the sanitized URL path
	finalPath := filepath.Join(rootPath, safePath)

	return finalPath, nil
}

func SaveToFile(rootPath string, s *snapshot.Snapshot, done chan struct{}) {
	parentPath := path.Join(rootPath, s.URL.Hostname)
	urlPath := s.URL.Path
	// If path is empty, add `index.gmi` as the file to save
	if urlPath == "" || urlPath == "." {
		urlPath = "index.gmi"
	}
	// If path ends with '/' then add index.gmi for the
	// directory to be created.
	if strings.HasSuffix(urlPath, "/") {
		urlPath = strings.Join([]string{urlPath, "index.gmi"}, "")
	}

	finalPath, err := calcFilePath(parentPath, urlPath)
	if err != nil {
		logging.LogError("GeminiError saving %s: %w", s.URL, err)
		return
	}
	// Ensure the directory exists
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		logging.LogError("Failed to create directory: %w", err)
		return
	}
	if s.MimeType.Valid && s.MimeType.String == "text/gemini" {
		err = os.WriteFile(finalPath, (*s).Data.V, 0o666)
	} else {
		err = os.WriteFile(finalPath, []byte((*s).GemText.String), 0o666)
	}
	if err != nil {
		logging.LogError("GeminiError saving %s: %w", s.URL.Full, err)
	}
	close(done)
}

func ReadLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %s", err))
	}
	lines := strings.Split(string(data), "\n")
	// remove last line if empty
	// (happens when file ends with '\n')
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
