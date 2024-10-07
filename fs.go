package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	fmt.Printf("%s %s\n", urlPath, cleanPath)
	// Safe check to prevent directory traversal
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid URL path: contains directory traversal")
	}

	// Sanitize the path by encoding invalid characters
	safePath := sanitizePath(cleanPath)

	// Join the root path and the sanitized URL path
	finalPath := filepath.Join(rootPath, safePath)

	// Ensure the directory exists
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directories: %v", err)
	}

	return finalPath, nil
}

func SaveResult(rootPath string, s *Snapshot) {
	urlPath := s.Url.Path
	if urlPath == "" || urlPath == "/" {
		urlPath = fmt.Sprintf("%s/index.gmi", s.Url.Hostname)
	}
	filepath, err := calcFilePath(rootPath, urlPath)
	if err != nil {
		LogError("Error saving %s: %w", s.Url, err)
		return
	}
	//	err = os.WriteFile(filepath, []byte(SnapshotToJSON(*s)), 0666)
	err = os.WriteFile(filepath, []byte((*s).Data), 0666)
	if err != nil {
		LogError("Error saving %s: %w", s.Url.Full, err)
	}
	LogInfo("[%s] Saved to %s", s.Url.Full, filepath)
}
