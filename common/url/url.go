package url

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"gemini-grc/errors"
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
		return errors.NewFatalError(fmt.Errorf("database scan error: expected string, got %T", value))
	}
	parsedURL, err := ParseURL(b, "", false)
	if err != nil {
		return err
	}
	*u = *parsedURL
	return nil
}

func (u URL) String() string {
	return u.Full
}

func (u URL) StringNoDefaultPort() string {
	if IsGeminiUrl(u.String()) {
		if u.Port == 1965 {
			return fmt.Sprintf("%s://%s%s", u.Protocol, u.Hostname, u.Path)
		}
	} else {
		if u.Port == 70 {
			return fmt.Sprintf("%s://%s%s", u.Protocol, u.Hostname, u.Path)
		}
	}
	return u.Full
}

func (u URL) Value() (driver.Value, error) {
	if u.Full == "" {
		return nil, nil
	}
	return u.Full, nil
}

func IsGeminiUrl(url string) bool {
	return strings.HasPrefix(url, "gemini://")
}

func IsGopherURL(s string) bool {
	return strings.HasPrefix(s, "gopher://")
}

func ParseURL(input string, descr string, normalize bool) (*URL, error) {
	var u *url.URL
	var err error
	if normalize {
		u, err = NormalizeURL(input)
		if err != nil {
			return nil, err
		}
	} else {
		u, err = url.Parse(input)
		if err != nil {
			return nil, errors.NewError(fmt.Errorf("error parsing URL: %w: %s", err, input))
		}
	}
	protocol := u.Scheme
	hostname := u.Hostname()
	strPort := u.Port()
	// urlPath := u.EscapedPath()
	urlPath := u.Path
	if strPort == "" {
		if u.Scheme == "gemini" {
			strPort = "1965" // default Gemini port
		} else {
			strPort = "70" // default Gopher port
		}
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, errors.NewError(fmt.Errorf("error parsing URL: %w: %s", err, input))
	}
	full := fmt.Sprintf("%s://%s:%d%s", protocol, hostname, port, urlPath)
	// full field should also contain query params and url fragments
	if u.RawQuery != "" {
		full += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		full += "#" + u.Fragment
	}
	return &URL{Protocol: protocol, Hostname: hostname, Port: port, Path: urlPath, Descr: descr, Full: full}, nil
}

// DeriveAbsoluteURL converts a (possibly) relative
// URL to an absolute one. Used primarily to calculate
// the full redirection URL target from a response header.
func DeriveAbsoluteURL(currentURL URL, input string) (*URL, error) {
	// If target URL is absolute, return just it
	if strings.Contains(input, "://") {
		return ParseURL(input, "", true)
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
	return ParseURL(strURL, "", true)
}

// NormalizeURL takes a URL string and returns a normalized version
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
		return nil, errors.NewError(fmt.Errorf("error normalizing URL: %w: %s", err, rawURL))
	}
	if u.Scheme == "" {
		return nil, errors.NewError(fmt.Errorf("error normalizing URL: No scheme: %s", rawURL))
	}
	if u.Host == "" {
		return nil, errors.NewError(fmt.Errorf("error normalizing URL: No host: %s", rawURL))
	}

	// Convert scheme to lowercase
	u.Scheme = strings.ToLower(u.Scheme)

	// Convert hostname to lowercase
	if u.Host != "" {
		u.Host = strings.ToLower(u.Host)
	}

	// remove default ports
	if u.Port() != "" {
		switch {
		case u.Scheme == "http" && u.Port() == "80":
			u.Host = u.Hostname()
		case u.Scheme == "https" && u.Port() == "443":
			u.Host = u.Hostname()
		case u.Scheme == "gemini" && u.Port() == "1965":
			u.Host = u.Hostname()
		case u.Scheme == "gopher" && u.Port() == "70":
			u.Host = u.Hostname()
		}
	}

	// Handle path normalization while preserving trailing slash
	if u.Path != "" {
		// Check if there was a trailing slash before cleaning
		hadTrailingSlash := strings.HasSuffix(u.Path, "/")

		u.Path = path.Clean(u.EscapedPath())
		// If path was "/", path.Clean() will return "."
		if u.Path == "." {
			u.Path = "/"
		} else if hadTrailingSlash && u.Path != "/" {
			// Restore trailing slash if it existed and path isn't just "/"
			u.Path += "/"
		}
	}

	// Properly escape the path, but only for unescaped parts
	parts := strings.Split(u.Path, "/")
	for i, part := range parts {
		// Try to unescape to check if it's already escaped
		unescaped, err := url.PathUnescape(part)
		if err != nil || unescaped == part {
			// Part is not escaped, so escape it
			parts[i] = url.PathEscape(part)
		}
		// If already escaped, leave as is
	}
	u.Path = strings.Join(parts, "/")

	// remove trailing fragment if empty
	if u.Fragment == "" {
		u.Fragment = ""
	}

	// remove trailing query if empty
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
	// Split URL into parts (protocol, host, p)
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

	// Split host and p
	parts = strings.SplitN(remainder, "/", 2)
	host := parts[0]
	if len(parts) == 1 {
		return protocol + "://" + host
	}

	// Escape the path portion
	escapedPath := url.PathEscape(parts[1])

	// Reconstruct the URL
	return protocol + "://" + host + "/" + escapedPath
}

// TrimTrailingPathSlash trims trailing slash and handles empty path
func TrimTrailingPathSlash(path string) string {
	// Handle empty path (e.g., "http://example.com" -> treat as root)
	if path == "" {
		return "/"
	}

	// Trim trailing slash while preserving root slash
	path = strings.TrimSuffix(path, "/")
	if path == "" { // This happens if path was just "/"
		return "/"
	}
	return path
}

// ExtractRedirectTargetFromHeader returns the redirection
// URL by parsing the header (or error message)
func ExtractRedirectTargetFromHeader(currentURL URL, input string) (*URL, error) {
	// \d+ - matches one or more digits
	// \s+ - matches one or more whitespace
	// ([^\r]+) - captures everything until it hits a \r (or end of string)
	pattern := `\d+\s+([^\r]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return nil, errors.NewError(fmt.Errorf("error extracting redirect target from string %s", input))
	}
	newURL, err := DeriveAbsoluteURL(currentURL, matches[1])
	if err != nil {
		return nil, err
	}
	return newURL, nil
}
