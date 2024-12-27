package gemini

import (
	"testing"
)

// Test for input: `20 text/gemini`
func TestGetMimeTypeAndLang1(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/gemini")
	if code != 20 || mimeType != "text/gemini" || lang != "" {
		t.Errorf("Expected (20, 'text/gemini', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang11(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/gemini\n")
	if code != 20 || mimeType != "text/gemini" || lang != "" {
		t.Errorf("Expected (20, 'text/gemini', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang12(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/plain; charset=utf-8")
	if code != 20 || mimeType != "text/plain" || lang != "utf-8" {
		t.Errorf("Expected (20, 'text/plain', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang13(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/gemini; charset=utf-8")
	if code != 20 || mimeType != "text/gemini" || lang != "utf-8" {
		t.Errorf("Expected (20, 'text/plain', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetTypeAndLang2(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/gemini charset=en")
	if code != 20 || mimeType != "text/gemini" || lang != "en" {
		t.Errorf("Expected (20, 'text/gemini', 'en'), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetTypeAndLang21(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("20 text/gemini lang=en")
	if code != 20 || mimeType != "text/gemini" || lang != "en" {
		t.Errorf("Expected (20, 'text/gemini', 'en'), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang3(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("31 gemini://redirect.to/page")
	if code != 31 || mimeType != "" || lang != "" {
		t.Errorf("Expected (20, '', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang4(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("aaafdasdasd")
	if code != 0 || mimeType != "" || lang != "" {
		t.Errorf("Expected (0, '', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}

func TestGetMimeTypeAndLang5(t *testing.T) {
	t.Parallel()
	code, mimeType, lang := getMimeTypeAndLang("")
	if code != 0 || mimeType != "" || lang != "" {
		t.Errorf("Expected (0, '', ''), got (%d, '%s', '%s')", code, mimeType, lang)
	}
}
