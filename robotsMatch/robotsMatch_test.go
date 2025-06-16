package robotsMatch

import (
	"context"
	"sync"
	"testing"

	"gemini-grc/config"
)

func TestInitializeShutdown(t *testing.T) {
	err := Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}

	err = Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}

func TestRobotMatch_EmptyCache(t *testing.T) {
	// This test doesn't actually connect to gemini URLs due to the complexity
	// of mocking the gemini client, but tests the caching behavior when no
	// robots.txt is found (empty cache case)
	config.CONFIG.ResponseTimeout = 5

	// Clear the cache before testing
	RobotsCache = sync.Map{}

	// For empty cache or DNS errors, RobotMatch should return false (allow the URL) without an error
	ctx := context.Background()
	blocked := RobotMatch(ctx, "gemini://nonexistent.example.com/")

	// The URL should be allowed (not blocked) when robots.txt can't be fetched
	if blocked {
		t.Errorf("Expected URL to be allowed when robots.txt can't be fetched")
	}
}
