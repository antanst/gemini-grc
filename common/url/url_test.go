package url

import (
	"reflect"
	"testing"
)

func TestURLOperations(t *testing.T) {
	t.Parallel()

	t.Run("ParseURL", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			input    string
			base     string
			absolute bool
			want     string
			wantErr  bool
		}{
			{
				name:     "parse CGI URL",
				input:    "gemini://caolan.uk/cgi-bin/weather.py/wxfcs/3162",
				base:     "",
				absolute: true,
				want:     "gemini://caolan.uk:1965/cgi-bin/weather.py/wxfcs/3162",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				parsed, err := ParseURL(tt.input, tt.base, tt.absolute)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr {
					value, _ := parsed.Value()
					if value != tt.want {
						t.Errorf("ParseURL() = %v, want %v", value, tt.want)
					}
				}
			})
		}
	})

	t.Run("DeriveAbsoluteURL", func(t *testing.T) {
		t.Parallel()

		baseURL := URL{
			Protocol: "gemini",
			Hostname: "smol.gr",
			Port:     1965,
			Path:     "/a/b",
			Descr:    "Nothing",
			Full:     "gemini://smol.gr:1965/a/b",
		}

		tests := []struct {
			name     string
			current  URL
			input    string
			expected *URL
		}{
			{
				name:    "absolute URL input",
				current: baseURL,
				input:   "gemini://a.b/c",
				expected: &URL{
					Protocol: "gemini",
					Hostname: "a.b",
					Port:     1965,
					Path:     "/c",
					Full:     "gemini://a.b:1965/c",
				},
			},
			{
				name:    "absolute path input",
				current: baseURL,
				input:   "/c",
				expected: &URL{
					Protocol: "gemini",
					Hostname: "smol.gr",
					Port:     1965,
					Path:     "/c",
					Full:     "gemini://smol.gr:1965/c",
				},
			},
			{
				name:    "relative path input",
				current: baseURL,
				input:   "c/d",
				expected: &URL{
					Protocol: "gemini",
					Hostname: "smol.gr",
					Port:     1965,
					Path:     "/a/b/c/d",
					Full:     "gemini://smol.gr:1965/a/b/c/d",
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				output, err := DeriveAbsoluteURL(tt.current, tt.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(output, tt.expected) {
					t.Errorf("got %#v, want %#v", output, tt.expected)
				}
			})
		}
	})

	t.Run("CheckAndUpdateNormalizedURL", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "with trailing slash",
				input:    "gemini://uscoffings.net/retro-computing/magazines/",
				expected: "gemini://uscoffings.net/retro-computing/magazines/",
			},
			{
				name:     "without trailing slash",
				input:    "gemini://uscoffings.net/retro-computing/magazines",
				expected: "gemini://uscoffings.net/retro-computing/magazines",
			},
			{
				name:     "multiple slashes",
				input:    "gemini://uscoffings.net/retro-computing/////////a///magazines",
				expected: "gemini://uscoffings.net/retro-computing/a/magazines",
			},
			{
				name:     "root with trailing slash",
				input:    "gemini://uscoffings.net/",
				expected: "gemini://uscoffings.net/",
			},
			{
				name:     "root without trailing slash",
				input:    "gemini://uscoffings.net",
				expected: "gemini://uscoffings.net",
			},
			{
				name:     "path with trailing slash",
				input:    "gemini://uscoffings.net/a/",
				expected: "gemini://uscoffings.net/a/",
			},
			{
				name:     "path without trailing slash",
				input:    "gemini://uscoffings.net/a",
				expected: "gemini://uscoffings.net/a",
			},
			{
				name:     "with dot segments",
				input:    "gemini://uscoffings.net/retro-computing/./././////a///magazines",
				expected: "gemini://uscoffings.net/retro-computing/a/magazines",
			},
			{
				name:     "with default port",
				input:    "gemini://uscoffings.net:1965/a",
				expected: "gemini://uscoffings.net/a",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				normalized, err := NormalizeURL(tt.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				output := normalized.String()
				if output != tt.expected {
					t.Errorf("got %#v, want %#v", output, tt.expected)
				}
			})
		}
	})
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with non-default port",
			input:    "gemini://chat.gemini.lehmann.cx:11965/",
			expected: "gemini://chat.gemini.lehmann.cx:11965/",
		},
		{
			name:     "URL with query parameters",
			input:    "gemini://chat.gemini.lehmann.cx:11965/index?a=1&b=c",
			expected: "gemini://chat.gemini.lehmann.cx:11965/index?a=1&b=c",
		},
		{
			name:     "URL with fragment",
			input:    "gemini://chat.gemini.lehmann.cx:11965/index#1",
			expected: "gemini://chat.gemini.lehmann.cx:11965/index#1",
		},
		{
			name:     "URL with CGI script and query",
			input:    "gemini://gemi.dev/cgi-bin/xkcd.cgi?1494",
			expected: "gemini://gemi.dev/cgi-bin/xkcd.cgi?1494",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel testing
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			normalized, err := NormalizeURL(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := normalized.String()
			if output != tt.expected {
				t.Errorf("got %#v, want %#v", output, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string // URL string to parse
		expected string // Expected normalized path
	}{
		// Basic cases
		{
			name:     "empty_path",
			input:    "http://example.com",
			expected: "",
		},
		{
			name:     "root_path",
			input:    "http://example.com/",
			expected: "/",
		},
		{
			name:     "single_trailing_slash",
			input:    "http://example.com/test/",
			expected: "/test/",
		},
		{
			name:     "no_trailing_slash",
			input:    "http://example.com/test",
			expected: "/test",
		},

		// Edge cases with slashes
		{
			name:     "multiple_trailing_slashes",
			input:    "http://example.com/test//",
			expected: "/test/",
		},
		{
			name:     "multiple_consecutive_slashes",
			input:    "http://example.com//test//",
			expected: "/test/",
		},
		{
			name:     "only_slashes",
			input:    "http://example.com////",
			expected: "/",
		},

		// Encoded characters
		{
			name:     "encoded_spaces",
			input:    "http://example.com/foo%20bar/",
			expected: "/foo%20bar/",
		},
		{
			name:     "encoded_special_chars",
			input:    "http://example.com/foo%2Fbar/",
			expected: "/foo%2Fbar/",
		},

		// Query parameters and fragments
		{
			name:     "with_query_parameters",
			input:    "http://example.com/path?query=param",
			expected: "/path",
		},
		{
			name:     "with_fragment",
			input:    "http://example.com/path#fragment",
			expected: "/path",
		},
		{
			name:     "with_both_query_and_fragment",
			input:    "http://example.com/path?query=param#fragment",
			expected: "/path",
		},

		// Unicode paths
		{
			name:     "unicode_characters",
			input:    "http://example.com/über/path/",
			expected: "/%C3%BCber/path/",
		},
		{
			name:     "unicode_encoded",
			input:    "http://example.com/%C3%BCber/path/",
			expected: "/%C3%BCber/path/",
		},

		// Weird but valid cases
		{
			name:     "dot_in_path",
			input:    "http://example.com/./path/",
			expected: "/path/",
		},
		{
			name:     "double_dot_in_path",
			input:    "http://example.com/../path/",
			expected: "/path/",
		},
		{
			name:     "mixed_case",
			input:    "http://example.com/PaTh/",
			expected: "/PaTh/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := ParseURL(tt.input, "", true)
			if err != nil {
				t.Fatalf("Failed to parse URL %q: %v", tt.input, err)
			}

			result := u.Path
			if result != tt.expected {
				t.Errorf("Input: %s\nExpected: %q\nGot: %q",
					u.Path, tt.expected, result)
			}
		})
	}
}

func TestExtractRedirectTargetFullURL(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://smol.gr", "", true)
	input := "redirect: 31 gemini://target.gr"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	expected := "gemini://target.gr:1965"
	if err != nil || (result.String() != expected) {
		t.Errorf("fail: Expected %s got %s", expected, result)
	}
}

func TestExtractRedirectTargetFullURLSlash(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://smol.gr", "", true)
	input := "redirect: 31 gemini://target.gr/"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	expected := "gemini://target.gr:1965/"
	if err != nil || (result.String() != expected) {
		t.Errorf("fail: Expected %s got %s", expected, result)
	}
}

func TestExtractRedirectTargetRelativeURL(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://smol.gr", "", true)
	input := "redirect: 31 /a/b"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	if err != nil || (result.String() != "gemini://smol.gr:1965/a/b") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetRelativeURL2(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://nox.im:1965", "", true)
	input := "redirect: 31 ./"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	if err != nil || (result.String() != "gemini://nox.im:1965/") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetRelativeURL3(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://status.zvava.org:1965", "", true)
	input := "redirect: 31 index.gmi"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	if err != nil || (result.String() != "gemini://status.zvava.org:1965/index.gmi") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetWrong(t *testing.T) {
	t.Parallel()
	currentURL, _ := ParseURL("gemini://smol.gr", "", true)
	input := "redirect: 31"
	result, err := ExtractRedirectTargetFromHeader(*currentURL, input)
	if result != nil || err == nil {
		t.Errorf("fail: result should be nil, err is %s", err)
	}
}
