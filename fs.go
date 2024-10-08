package main

import (
	"fmt"
	"net/url"
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
