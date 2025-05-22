package whiteList

import (
	"os"
	"regexp"
	"testing"

	"gemini-grc/config"
)

func TestIsWhitelisted(t *testing.T) {
	// Set up a test whitelist
	whitelist = []regexp.Regexp{
		*regexp.MustCompile(`^gemini://example\.com`),
		*regexp.MustCompile(`^gemini://test\.org/path`),
	}

	testCases := []struct {
		url      string
		expected bool
	}{
		{"gemini://example.com", true},
		{"gemini://example.com/path", true},
		{"gemini://test.org", false},
		{"gemini://test.org/path", true},
		{"gemini://test.org/path/subpath", true},
		{"gemini://other.site", false},
	}

	for _, tc := range testCases {
		result := IsWhitelisted(tc.url)
		if result != tc.expected {
			t.Errorf("IsWhitelisted(%s) = %v, want %v", tc.url, result, tc.expected)
		}
	}
}

func TestLoadWhitelist(t *testing.T) {
	// Create a temporary whitelist file
	content := `# This is a test whitelist
^gemini://example\.com
^gemini://test\.org/path
`
	tmpfile, err := os.CreateTemp("", "whitelist")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset whitelist
	whitelist = nil

	// Set up configuration to use the temporary file
	oldPath := config.CONFIG.WhitelistPath
	config.CONFIG.WhitelistPath = tmpfile.Name()
	defer func() {
		config.CONFIG.WhitelistPath = oldPath
	}()

	// Load whitelist from the file
	err = loadWhitelist(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadWhitelist() error = %v", err)
	}

	// Check if whitelist was loaded correctly
	if len(whitelist) != 2 {
		t.Errorf("loadWhitelist() loaded %d entries, want 2", len(whitelist))
	}

	// Test a whitelisted URL
	if !IsWhitelisted("gemini://example.com") {
		t.Error("IsWhitelisted(\"gemini://example.com\") = false, want true")
	}

	// Test a URL in a whitelisted path
	if !IsWhitelisted("gemini://test.org/path/subpage.gmi") {
		t.Error("IsWhitelisted(\"gemini://test.org/path/subpage.gmi\") = false, want true")
	}
}
