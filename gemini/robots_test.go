package gemini

import (
	"reflect"
	"testing"
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
		"gemini://example.com/admin/",
	}

	result := ParseRobotsTxt(input, "example.com")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ParseRobotsTxt() = %v, want %v", result, expected)
	}
}

func TestParseRobotsTxtEmpty(t *testing.T) {
	input := ``

	result := ParseRobotsTxt(input, "example.com")

	if len(result) != 0 {
		t.Errorf("ParseRobotsTxt() = %v, want empty []string", result)
	}
}
