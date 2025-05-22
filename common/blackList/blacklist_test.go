package blackList

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"gemini-grc/config"
)

func TestIsBlacklisted(t *testing.T) {
	// Save original blacklist and whitelist to restore after test
	originalBlacklist := blacklist
	defer func() {
		blacklist = originalBlacklist
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
				blacklist = []regexp.Regexp{}
			},
			url:      "https://example.com",
			expected: false,
		},
		{
			name: "exact hostname match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "example.com",
			expected: true,
		},
		{
			name: "hostname in URL match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name: "partial hostname match",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://safe-example.com",
			expected: true,
		},
		{
			name: "full URL match",
			setup: func() {
				regex, _ := regexp.Compile(`https://example\.com/bad-path`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/bad-path",
			expected: true,
		},
		{
			name: "path match",
			setup: func() {
				regex, _ := regexp.Compile("/malicious-path")
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/malicious-path",
			expected: true,
		},
		{
			name: "subdomain match with word boundary",
			setup: func() {
				regex, _ := regexp.Compile(`bad\.example\.com`)
				blacklist = []regexp.Regexp{*regex}
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
				blacklist = []regexp.Regexp{*regex1, *regex2, *regex3}
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
				blacklist = []regexp.Regexp{*regex1, *regex2, *regex3}
			},
			url:      "https://example.com/safe/data",
			expected: false,
		},
		{
			name: "pattern with wildcard",
			setup: func() {
				regex, _ := regexp.Compile(`.*\.evil\.com`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://subdomain.evil.com/path",
			expected: true,
		},
		{
			name: "pattern with special characters",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com/path\?id=[0-9]+`)
				blacklist = []regexp.Regexp{*regex}
			},
			url:      "https://example.com/path?id=12345",
			expected: true,
		},
		{
			name: "unicode character support",
			setup: func() {
				regex, _ := regexp.Compile(`example\.com/[\p{L}]+`)
				blacklist = []regexp.Regexp{*regex}
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

// TestBlacklistLoading tests that the blacklist loading logic works with a mock blacklist file
func TestBlacklistLoading(t *testing.T) {
	// Save original blacklist and config
	originalBlacklist := blacklist
	originalConfigPath := config.CONFIG.BlacklistPath
	defer func() {
		blacklist = originalBlacklist
		config.CONFIG.BlacklistPath = originalConfigPath
	}()

	// Create a temporary blacklist file with known patterns
	tmpFile, err := os.CreateTemp("", "mock-blacklist-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some test patterns to the mock blacklist file
	mockBlacklistContent := `# Mock blacklist file for testing
/git/
/.git/
/cgit/
gemini://git\..*$
gemini://.*/git/.*
gopher://.*/git/.*
.*/(commit|blob|tree)/.*
.*/[0-9a-f]{7,40}$
`
	if err := os.WriteFile(tmpFile.Name(), []byte(mockBlacklistContent), 0o644); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Configure and load the mock blacklist
	blacklist = nil
	config.CONFIG.BlacklistPath = tmpFile.Name()
	err = Initialize()
	if err != nil {
		t.Fatalf("Failed to load mock blacklist: %v", err)
	}

	// Count the number of non-comment, non-empty lines to verify loading
	lineCount := 0
	for _, line := range strings.Split(mockBlacklistContent, "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			lineCount++
		}
	}

	if len(blacklist) != lineCount {
		t.Errorf("Expected %d patterns to be loaded, got %d", lineCount, len(blacklist))
	}

	// Verify some sample URLs against our known patterns
	testURLs := []struct {
		url      string
		expected bool
		desc     string
	}{
		{"gemini://example.com/git/repo", true, "git repository"},
		{"gemini://git.example.com", true, "git subdomain"},
		{"gemini://example.com/cgit/repo", true, "cgit repository"},
		{"gemini://example.com/repo/commit/abc123", true, "git commit"},
		{"gemini://example.com/123abc7", true, "commit hash at path end"},
		{"gopher://example.com/1/git/repo", true, "gopher git repository"},
		{"gemini://example.com/normal/page.gmi", false, "normal gemini page"},
		{"gemini://example.com/project/123abc", false, "hash not at path end"},
	}

	for _, tt := range testURLs {
		result := IsBlacklisted(tt.url)
		if result != tt.expected {
			t.Errorf("With mock blacklist, IsBlacklisted(%q) = %v, want %v", tt.url, result, tt.expected)
		}
	}
}

func TestLoadBlacklist(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := blacklist
	originalConfigPath := config.CONFIG.BlacklistPath
	defer func() {
		blacklist = originalBlacklist
		config.CONFIG.BlacklistPath = originalConfigPath
	}()

	// Create a temporary blacklist file for testing
	tmpFile, err := os.CreateTemp("", "blacklist-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Test cases for Initialize
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
			blacklist = nil

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
			err := Initialize()

			// Check results
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(blacklist) != tt.expectedLen {
				t.Errorf("Initialize() loaded %d entries, want %d", len(blacklist), tt.expectedLen)
			}
		})
	}
}

// TestGitPatterns tests the blacklist patterns specifically for Git repositories
func TestGitPatterns(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := blacklist
	defer func() {
		blacklist = originalBlacklist
	}()

	// Create patterns similar to those in the blacklist.txt file
	patterns := []string{
		"/git/",
		"/.git/",
		"/cgit/",
		"/gitweb/",
		"/gitea/",
		"/scm/",
		".*/(commit|blob|tree|tag|diff|blame|log|raw)/.*",
		".*/(commits|objects|refs|branches|tags)/.*",
		".*/[0-9a-f]{7,40}$",
		"gemini://git\\..*$",
		"gemini://.*/git/.*",
		"gemini://.*\\.git/.*",
		"gopher://.*/git/.*",
	}

	// Compile and set up the patterns
	blacklist = []regexp.Regexp{}
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
		}
		blacklist = append(blacklist, *regex)
	}

	// Test URLs against git-related patterns
	tests := []struct {
		url      string
		expected bool
		desc     string
	}{
		// Git paths
		{"gemini://example.com/git/", true, "basic git path"},
		{"gemini://example.com/.git/", true, "hidden git path"},
		{"gemini://example.com/cgit/", true, "cgit path"},
		{"gemini://example.com/gitweb/", true, "gitweb path"},
		{"gemini://example.com/gitea/", true, "gitea path"},
		{"gemini://example.com/scm/", true, "scm path"},

		// Git operations
		{"gemini://example.com/repo/commit/abc123", true, "commit path"},
		{"gemini://example.com/repo/blob/main/README.md", true, "blob path"},
		{"gemini://example.com/repo/tree/master", true, "tree path"},
		{"gemini://example.com/repo/tag/v1.0", true, "tag path"},

		// Git internals
		{"gemini://example.com/repo/commits/", true, "commits path"},
		{"gemini://example.com/repo/objects/", true, "objects path"},
		{"gemini://example.com/repo/refs/heads/main", true, "refs path"},

		// Git hashes
		{"gemini://example.com/commit/a1b2c3d", true, "short hash"},
		{"gemini://example.com/commit/a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", true, "long hash"},

		// Git domains
		{"gemini://git.example.com/", true, "git subdomain"},
		{"gemini://example.com/git/repo", true, "git directory"},
		{"gemini://example.com/project.git/", true, "git extension"},

		// Gopher protocol
		{"gopher://example.com/1/git/repo", true, "gopher git path"},

		// Non-matching URLs
		{"gemini://example.com/project/", false, "regular project path"},
		{"gemini://example.com/blog/", false, "blog path"},
		{"gemini://example.com/git-guide.gmi", false, "hyphenated word with git"},
		{"gemini://example.com/digital/", false, "word containing 'git'"},
		{"gemini://example.com/ab12cd3", true, "short hex string matches commit hash pattern"},
		{"gemini://example.com/ab12cdz", false, "alphanumeric string with non-hex chars won't match commit hash"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := IsBlacklisted(tt.url)
			if result != tt.expected {
				t.Errorf("IsBlacklisted(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestGeminiGopherPatterns tests the blacklist patterns specific to Gemini and Gopher protocols
func TestGeminiGopherPatterns(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := blacklist
	defer func() {
		blacklist = originalBlacklist
	}()

	// Create patterns for Gemini and Gopher
	patterns := []string{
		"gemini://badhost\\.com",
		"gemini://.*/cgi-bin/",
		"gemini://.*/private/",
		"gemini://.*\\.evil\\..*",
		"gopher://badhost\\.org",
		"gopher://.*/I/onlyfans/",
		"gopher://.*/[0-9]/(cgi|bin)/",
	}

	// Compile and set up the patterns
	blacklist = []regexp.Regexp{}
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
		}
		blacklist = append(blacklist, *regex)
	}

	// Test URLs against Gemini and Gopher patterns
	tests := []struct {
		url      string
		expected bool
		desc     string
	}{
		// Gemini URLs
		{"gemini://badhost.com/", true, "blacklisted gemini host"},
		{"gemini://badhost.com/page.gmi", true, "blacklisted gemini host with path"},
		{"gemini://example.com/cgi-bin/script.cgi", true, "gemini cgi-bin path"},
		{"gemini://example.com/private/docs", true, "gemini private path"},
		{"gemini://subdomain.evil.org", true, "gemini evil domain pattern"},
		{"gemini://example.com/public/docs", false, "safe gemini path"},
		{"gemini://goodhost.com/", false, "safe gemini host"},

		// Gopher URLs
		{"gopher://badhost.org/1/menu", true, "blacklisted gopher host"},
		{"gopher://example.org/I/onlyfans/image", true, "gopher onlyfans path"},
		{"gopher://example.org/1/cgi/script", true, "gopher cgi path"},
		{"gopher://example.org/1/bin/executable", true, "gopher bin path"},
		{"gopher://example.org/0/text", false, "safe gopher text"},
		{"gopher://goodhost.org/1/menu", false, "safe gopher host"},

		// Protocol distinction
		{"https://badhost.com/", false, "blacklisted host but wrong protocol"},
		{"http://example.com/cgi-bin/script.cgi", false, "bad path but wrong protocol"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := IsBlacklisted(tt.url)
			if result != tt.expected {
				t.Errorf("IsBlacklisted(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsBlacklistedIntegration(t *testing.T) {
	// Save original blacklist to restore after test
	originalBlacklist := blacklist
	originalBlacklistPath := config.CONFIG.BlacklistPath
	defer func() {
		blacklist = originalBlacklist
		config.CONFIG.BlacklistPath = originalBlacklistPath
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
	blacklist = nil
	config.CONFIG.BlacklistPath = tmpFile.Name()

	// Load the blacklist
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
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
