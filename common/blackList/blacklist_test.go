package blackList

import (
	"os"
	"regexp"
	"testing"

	"gemini-grc/config"
)

func TestIsBlacklisted(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := Blacklist
	defer func() {
		Blacklist = originalBlacklist
	}()

	tests := []struct {
		name     string
		setup    func()
		url      string
		expected bool
	}{
		{
			name: "empty blacklist",
			setup: func() {
				Blacklist = []regexp.Regexp{}
			},
			url:      "https://example.com",
			expected: false,
		},
		{
			name: "exact hostname match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "example.com",
			expected: true,
		},
		{
			name: "hostname in URL match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name: "partial hostname match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://safe-example.com",
			expected: true,
		},
		{
			name: "full URL match",
			setup: func() {
				regex, _ := regexp.Compile(`https://example\.com/bad-path`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/bad-path",
			expected: true,
		},
		{
			name: "path match",
			setup: func() {
				regex, _ := regexp.Compile("/malicious-path")
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/malicious-path",
			expected: true,
		},
		{
			name: "subdomain match with word boundary",
			setup: func() {
				regex, _ := regexp.Compile(`bad\.example\.com`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://bad.example.com/path",
			expected: true,
		},
		{
			name: "multiple patterns, one match",
			setup: func() {
				regex1, _ := regexp.Compile(`badsite\.com`)
				regex2, _ := regexp.Compile(`malicious\.org`)
				regex3, _ := regexp.Compile(`example\.com/sensitive`)
				Blacklist = []regexp.Regexp{*regex1, *regex2, *regex3}
			},
			url:      "https://example.com/sensitive/data",
			expected: true,
		},
		{
			name: "multiple patterns, no match",
			setup: func() {
				regex1, _ := regexp.Compile(`badsite\.com`)
				regex2, _ := regexp.Compile(`malicious\.org`)
				regex3, _ := regexp.Compile(`example\.com/sensitive`)
				Blacklist = []regexp.Regexp{*regex1, *regex2, *regex3}
			},
			url:      "https://example.com/safe/data",
			expected: false,
		},
		{
			name: "pattern with wildcard",
			setup: func() {
				regex, _ := regexp.Compile(`.*\.evil\.com`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://subdomain.evil.com/path",
			expected: true,
		},
		{
			name: "pattern with special characters",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com/path\?id=[0-9]+`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/path?id=12345",
			expected: true,
		},
		{
			name: "unicode character support",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com/[\p{L}]+`)
				Blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/café",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := IsBlacklisted(tt.url)
			if result != tt.expected {
				t.Errorf("IsBlacklisted(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestLoadBlacklist(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := Blacklist
	originalConfigPath := config.CONFIG.BlacklistPath
	defer func() {
		Blacklist = originalBlacklist
		config.CONFIG.BlacklistPath = originalConfigPath
	}()

	// Create a temporary blacklist file for testing
	tmpFile, err := os.CreateTemp("", "blacklist-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Test cases for LoadBlacklist
	tests := []struct {
		name           string
		blacklistLines []string
		configPath     string
		wantErr        bool
		expectedLen    int
	}{
		{
			name:           "empty path",
			blacklistLines: []string{},
			configPath:     "",
			wantErr:        false,
			expectedLen:    0,
		},
		{
			name:           "valid blacklist with comments",
			blacklistLines: []string{"example\\.com", "# This is a comment", "malicious\\.org"},
			configPath:     tmpFile.Name(),
			wantErr:        false,
			expectedLen:    2,
		},
		{
			name:           "invalid regex",
			blacklistLines: []string{"example\\.com", "[invalid regex"},
			configPath:     tmpFile.Name(),
			wantErr:        true,
			expectedLen:    0,
		},
		{
			name:           "nonexistent file",
			blacklistLines: []string{},
			configPath:     "nonexistent-file.txt",
			wantErr:        true,
			expectedLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset blacklist
			Blacklist = nil

			// Set config path
			config.CONFIG.BlacklistPath = tt.configPath

			// Write test data to file if needed
			if tt.configPath == tmpFile.Name() {
				content := ""
				for _, line := range tt.blacklistLines {
					content += line + "\n"
				}
				if err := os.WriteFile(tmpFile.Name(), []byte(content), 0o644); err != nil {
					t.Fatalf("Failed to write to temporary file: %v", err)
				}
			}

			// Call the function
			err := LoadBlacklist()

			// Check results
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadBlacklist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(Blacklist) != tt.expectedLen {
				t.Errorf("LoadBlacklist() loaded %d entries, want %d", len(Blacklist), tt.expectedLen)
			}
		})
	}
}

// TestIsBlacklistedIntegration tests the integration between LoadBlacklist and IsBlacklisted
func TestIsBlacklistedIntegration(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := Blacklist
	originalConfigPath := config.CONFIG.BlacklistPath
	defer func() {
		Blacklist = originalBlacklist
		config.CONFIG.BlacklistPath = originalConfigPath
	}()

	// Create a temporary blacklist file for testing
	tmpFile, err := os.CreateTemp("", "blacklist-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test patterns to the blacklist file
	blacklistContent := `# Test blacklist file
example\.com
malicious\.org
/phishing
.*\.evil\.com
\w+@spam\.com
`
	if err := os.WriteFile(tmpFile.Name(), []byte(blacklistContent), 0o644); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Set up the test
	Blacklist = nil
	config.CONFIG.BlacklistPath = tmpFile.Name()

	// Load the blacklist
	if err := LoadBlacklist(); err != nil {
		t.Fatalf("LoadBlacklist() failed: %v", err)
	}

	// Test URLs against the loaded blacklist
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"https://safe-site.com", false},
		{"https://malicious.org/path", true},
		{"https://example.org/phishing", true},
		{"https://subdomain.evil.com", true},
		{"https://safe-site.com/safe-path", false},
		{"mailto:user@spam.com", true},
	}

	for _, tt := range tests {
		result := IsBlacklisted(tt.url)
		if result != tt.expected {
			t.Errorf("IsBlacklisted(%q) = %v, want %v", tt.url, result, tt.expected)
		}
	}
}
