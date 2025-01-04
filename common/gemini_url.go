package common

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type URL struct {
	Protocol string `json:"protocol,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
	Path     string `json:"path,omitempty"`
	Descr    string `json:"descr,omitempty"`
	Full     string `json:"full,omitempty"`
}

func (u *URL) Scan(value interface{}) error {
	if value == nil {
		// Clear the fields in the current GeminiUrl object (not the pointer itself)
		*u = URL{}
		return nil
	}
	b, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan GeminiUrl: expected string, got %T", value)
	}
	parsedURL, err := ParseURLNoNormalize(b, "")
	if err != nil {
		err = fmt.Errorf("failed to scan GeminiUrl %s: %v", b, err)
		return err
	}
	*u = *parsedURL
	return nil
}

func (u URL) String() string {
	return u.Full
}

func (u URL) StringNoDefaultPort() string {
	if u.Port == 1965 {
		return fmt.Sprintf("%s://%s%s", u.Protocol, u.Hostname, u.Path)
	}
	return u.Full
}

func (u URL) Value() (driver.Value, error) {
	if u.Full == "" {
		return nil, nil
	}
	return u.Full, nil
}

func ParseURLNoNormalize(input string, descr string) (*URL, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("%w: Input %s URL Parse Error: %w", ErrURLParse, input, err)
	}
	if u.Scheme != "gemini" {
		return nil, fmt.Errorf("%w: URL scheme '%s' is not supported", ErrURLNotGemini, u.Scheme)
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	strPort := u.Port()
	urlPath := u.Path
	if strPort == "" {
		strPort = "1965"
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, fmt.Errorf("%w: Input %s GeminiError %w", ErrURLParse, input, err)
	}
	full := fmt.Sprintf("%s://%s:%d%s", protocol, hostname, port, urlPath)
	return &URL{Protocol: protocol, Hostname: hostname, Port: port, Path: urlPath, Descr: descr, Full: full}, nil
}

func ParseURL(input string, descr string) (*URL, error) {
	u, err := NormalizeURL(input)
	if err != nil {
		return nil, fmt.Errorf("%w: Input %s URL Parse Error: %w", ErrURLParse, input, err)
	}
	if u.Scheme != "gemini" {
		return nil, fmt.Errorf("%w: URL scheme '%s' is not supported", ErrURLNotGemini, u.Scheme)
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	strPort := u.Port()
	urlPath := u.Path
	if strPort == "" {
		strPort = "1965"
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, fmt.Errorf("%w: Input %s GeminiError %w", ErrURLParse, input, err)
	}
	full := fmt.Sprintf("%s://%s:%d%s", protocol, hostname, port, urlPath)
	return &URL{Protocol: protocol, Hostname: hostname, Port: port, Path: urlPath, Descr: descr, Full: full}, nil
}

// DeriveAbsoluteURL converts a (possibly) relative
// URL to an absolute one. Used primarily to calculate
// the full redirection URL target from a response header.
func DeriveAbsoluteURL(currentURL URL, input string) (*URL, error) {
	// If target URL is absolute, return just it
	if strings.Contains(input, "://") {
		return ParseURL(input, "")
	}
	// input is a relative path. Clean it and construct absolute.
	var newPath string
	// Handle weird cases found in the wild
	if strings.HasPrefix(input, "/") {
		newPath = path.Clean(input)
	} else if input == "./" || input == "." {
		newPath = path.Join(currentURL.Path, "/")
	} else {
		newPath = path.Join(currentURL.Path, "/", path.Clean(input))
	}
	strURL := fmt.Sprintf("%s://%s:%d%s", currentURL.Protocol, currentURL.Hostname, currentURL.Port, newPath)
	return ParseURL(strURL, "")
}

// NormalizeURL takes a URL string and returns a normalized version.
// Normalized meaning:
// - Path normalization (removing redundant slashes, . and .. segments)
// - Proper escaping of special characters
// - Lowercase scheme and host
// - Removal of default ports
// - Empty path becomes "/"
func NormalizeURL(rawURL string) (*url.URL, error) {
	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrURLParse, err)
	}

	// Convert scheme to lowercase
	u.Scheme = strings.ToLower(u.Scheme)

	// Convert hostname to lowercase
	if u.Host != "" {
		u.Host = strings.ToLower(u.Host)
	}

	// Remove default ports
	if u.Port() != "" {
		switch {
		case u.Scheme == "http" && u.Port() == "80":
			u.Host = u.Hostname()
		case u.Scheme == "https" && u.Port() == "443":
			u.Host = u.Hostname()
		case u.Scheme == "gemini" && u.Port() == "1965":
			u.Host = u.Hostname()
		}
	}

	// Handle path normalization while preserving trailing slash
	if u.Path != "" {
		// Check if there was a trailing slash before cleaning
		hadTrailingSlash := strings.HasSuffix(u.Path, "/")

		u.Path = path.Clean(u.Path)
		// If path was "/", path.Clean() will return "."
		if u.Path == "." {
			u.Path = "/"
		} else if hadTrailingSlash && u.Path != "/" {
			// Restore trailing slash if it existed and path isn't just "/"
			u.Path += "/"
		}
	}

	// Properly escape the path
	// First split on '/' to avoid escaping them
	parts := strings.Split(u.Path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	u.Path = strings.Join(parts, "/")

	// Remove trailing fragment if empty
	if u.Fragment == "" {
		u.Fragment = ""
	}

	// Remove trailing query if empty
	if u.RawQuery == "" {
		u.RawQuery = ""
	}

	return u, nil
}

func EscapeURL(input string) string {
	// Only escape if not already escaped
	if strings.Contains(input, "%") && !strings.Contains(input, "% ") {
		return input
	}
	// Split URL into parts (protocol, host, path)
	parts := strings.SplitN(input, "://", 2)
	if len(parts) != 2 {
		return input
	}

	protocol := parts[0]
	remainder := parts[1]

	// If URL ends with just a slash, return as is
	if strings.HasSuffix(remainder, "/") && !strings.Contains(remainder[:len(remainder)-1], "/") {
		return input
	}

	// Split host and path
	parts = strings.SplitN(remainder, "/", 2)
	host := parts[0]
	if len(parts) == 1 {
		return protocol + "://" + host
	}

	path := parts[1]

	// Escape the path portion
	escapedPath := url.PathEscape(path)

	// Reconstruct the URL
	return protocol + "://" + host + "/" + escapedPath
}
