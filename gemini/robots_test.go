package gemini

import (
	"testing"
	"reflect"
)

func TestParseRobotsTxt(t *testing.T) {
	input := `User-agent: *
Disallow: /cgi-bin/wp.cgi/view
Disallow: /cgi-bin/wp.cgi/media
User-agent: googlebot
Disallow: /admin/`

	expected := []string{
		"gemini://example.com/cgi-bin/wp.cgi/view",
		"gemini://example.com/cgi-bin/wp.cgi/media",
	}

	result := ParseRobotsTxt(input, "example.com")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ParseRobotsTxt() = %v, want %v", result, expected)
	}
}
