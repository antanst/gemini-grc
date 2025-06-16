package seedList

import (
	"os"
	"testing"
)

func TestLoadSeedlist(t *testing.T) {
	// Create a temporary test file
	content := `# Test seed URLs
gemini://example.com/
gemini://test.com/

# Another comment
gemini://demo.org/`

	tmpFile, err := os.CreateTemp("", "seed_urls_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Reset global variable for test
	seedlist = nil

	// Test loading
	err = loadSeedlist(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load seedlist: %v", err)
	}

	// Verify content
	expected := []string{
		"gemini://example.com/",
		"gemini://test.com/",
		"gemini://demo.org/",
	}

	urls := GetSeedURLs()
	if len(urls) != len(expected) {
		t.Errorf("Expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("Expected URL %d to be %s, got %s", i, expected[i], url)
		}
	}
}

func TestGetSeedURLsEmptyList(t *testing.T) {
	// Reset global variable
	originalSeedlist := seedlist
	defer func() { seedlist = originalSeedlist }()

	seedlist = nil

	urls := GetSeedURLs()
	if len(urls) != 0 {
		t.Errorf("Expected empty list, got %d URLs", len(urls))
	}
}
