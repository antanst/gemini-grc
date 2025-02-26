package gopher

import (
	"net"
	"testing"

	"gemini-grc/common/errors"
	"gemini-grc/config"
	"github.com/stretchr/testify/assert"
)

func TestConstructPayloadFromPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Path with Gopher item type",
			input:    "/1/path/to/resource",
			expected: "/path/to/resource",
		},
		{
			name:     "Path with different item type",
			input:    "/0/another/path",
			expected: "/another/path",
		},
		{
			name:     "Path without item type but with leading slash",
			input:    "/simple/path",
			expected: "/simple/path",
		},
		{
			name:     "Path without item type and without leading slash",
			input:    "no/leading/slash",
			expected: "/no/leading/slash",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "Single character item type",
			input:    "/h/homepage",
			expected: "/homepage",
		},
		{
			name:     "Single slash",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Item type-looking path",
			input:    "/1",
			expected: "/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constructPayloadFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("constructPayloadFromPath(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseLinks(t *testing.T) {
	tests := []struct {
		name       string
		currentURL string
		input      string
		want       int // number of expected links
		wantErr    bool
	}{
		{
			name:       "Empty input",
			currentURL: "gopher://example.com:70",
			input:      "",
			want:       0,
			wantErr:    false,
		},
		{
			name:       "Single directory link",
			currentURL: "gopher://example.com:70",
			input:      "1About Us\t/about\texample.com",
			want:       1,
			wantErr:    false,
		},
		{
			name:       "Single text file link",
			currentURL: "gopher://example.com:70",
			input:      "0README\t/readme.txt\texample.com",
			want:       1,
			wantErr:    false,
		},
		{
			name:       "Multiple links of different types",
			currentURL: "gopher://example.com:70",
			input:      "1About Us\t/about\texample.com\n0README\t/readme.txt\texample.com\n1Contact\t/contact\texample.com",
			want:       3,
			wantErr:    false,
		},
		{
			name:       "Ignore non-linkable types",
			currentURL: "gopher://example.com:70",
			input:      "iInfo line\t/info\texample.com\n1Directory\t/dir\texample.com\n0Text\t/text.txt\texample.com",
			want:       2,
			wantErr:    false,
		},
		{
			name:       "Malformed lines",
			currentURL: "gopher://example.com:70",
			input:      "1Incomplete line\n0No tabs\n1Missing parts\t",
			want:       0,
			wantErr:    false,
		},
		{
			name:       "Mixed valid and invalid lines",
			currentURL: "gopher://example.com:70",
			input:      "1Valid link\t/valid\texample.com\n1Incomplete\t\n0Text file\t/text.txt\texample.com\n1Another valid\t/another\texample.com",
			want:       3,
			wantErr:    false,
		},
		{
			name:       "Absolute URLs",
			currentURL: "gopher://example.com:70",
			input:      "1External link\tgopher://external.com/path\texternal.com\n0Document\tgopher://other.com/doc.txt\tother.com",
			want:       2,
			wantErr:    false,
		},
		{
			name:       "With whitespace",
			currentURL: "gopher://example.com:70",
			input:      "  1Padded line  \t/padded\texample.com\n0Text file  \t/doc.txt\texample.com",
			want:       2,
			wantErr:    false,
		},
		{
			name:       "Special characters in paths",
			currentURL: "gopher://example.com:70",
			input:      "1Special chars\t/path with spaces\texample.com\n0Doc\t/über/päth.txt\texample.com",
			want:       2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGopherPageLinks(tt.input)
			assert.Equal(t, tt.want, len(got), "expected %d links, got %d", tt.want, len(got))
		})
	}
}

func TestCheckForError(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantError   bool
		errorPrefix string
	}{
		{
			name:        "No error",
			input:       "1Directory\t/dir\texample.com\n0Text\t/text.txt\texample.com",
			wantError:   false,
			errorPrefix: "",
		},
		{
			name:        "Simple error message",
			input:       "3Error: File not found\t\texample.com",
			wantError:   true,
			errorPrefix: "gopher error: 3Error: File not found",
		},
		{
			name:        "Error with multiple tabs",
			input:       "3File not found\t/error\texample.com\t70",
			wantError:   true,
			errorPrefix: "gopher error: 3File not found",
		},
		{
			name: "Error among valid entries",
			input: `1Welcome\t/welcome\texample.com
3Access denied\t\texample.com
0README\t/readme.txt\texample.com`,
			wantError:   false,
			errorPrefix: "",
		},
		{
			name:        "Error with no tabs",
			input:       "3Server is down for maintenance",
			wantError:   true,
			errorPrefix: "gopher error: 3Server is down for maintenance",
		},
		{
			name: "Multiple errors (should return first)",
			input: `3First error\t\texample.com
3Second error\t\texample.com`,
			wantError:   true,
			errorPrefix: "gopher error: 3First error",
		},
		{
			name:        "Error with whitespace",
			input:       "  3  Error with spaces  \t\texample.com",
			wantError:   true,
			errorPrefix: "gopher error: 3  Error with spaces",
		},
		{
			name:        "Empty input",
			input:       "",
			wantError:   false,
			errorPrefix: "",
		},
		{
			name:        "Just newlines",
			input:       "\n\n\n",
			wantError:   false,
			errorPrefix: "",
		},
		{
			name: "Error after empty lines",
			input: `

3Error after empty lines\t\texample.com`,
			wantError:   true,
			errorPrefix: "gopher error: 3Error after empty lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkForError(tt.input)

			if !tt.wantError {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorPrefix)
		})
	}
}

func TestConnectAndGetDataTimeout(t *testing.T) {
	// Start a test server that doesn't respond
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Accept the connection but don't respond
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Logf("Failed to accept connection: %v", err)
			return
		}
		defer conn.Close()

		// Keep the connection open without sending any data to simulate a timeout
		select {}
	}()

	// Construct the URL of our test server
	address := listener.Addr().String()
	testURL := "gopher://" + address + "/testpath"

	// Save original config values
	originalTimeout := config.CONFIG.ResponseTimeout
	originalMaxSize := config.CONFIG.MaxResponseSize

	// Set test config values
	config.CONFIG.ResponseTimeout = 1    // Set a very short timeout for this test
	config.CONFIG.MaxResponseSize = 1024 // Just for consistency, we won't reach this

	// Test the function
	_, err = connectAndGetData(testURL)

	// Reset config values
	config.CONFIG.ResponseTimeout = originalTimeout
	config.CONFIG.MaxResponseSize = originalMaxSize

	// Check if the error is due to timeout
	if err == nil {
		t.Error("Expected an error due to timeout, but got no error")
	} else if !errors.IsHostError(err) {
		t.Errorf("Expected a HostError, but got: %v", err)
	} else {
		// Here you might want to check if the specific error message contains 'timeout'
		// However, since we don't have the exact error string, we're checking the type
		t.Logf("Successfully timed out: %v", err)
	}
}
