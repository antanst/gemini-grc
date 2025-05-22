package gemini

import (
	"slices"
	"strings"
	"testing"

	"gemini-grc/common/snapshot"
)

func TestGetHeadersAndData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input       []byte
		header      string
		body        []byte
		expectError bool
	}{
		{[]byte("20 text/gemini\r\nThis is the body"), "20 text/gemini", []byte("This is the body"), false},
		{[]byte("20 text/gemini\nThis is the body"), "20 text/gemini", []byte("This is the body"), false},
		{[]byte("53 No proxying!\r\n"), "53 No proxying!", []byte(""), false},
		{[]byte("No header"), "", nil, true},
	}

	for _, test := range tests {
		header, body, err := getHeadersAndData(test.input)

		if test.expectError && err == nil {
			t.Errorf("Expected error, got nil for input: %s", test.input)
		}

		if !test.expectError && err != nil {
			t.Errorf("Unexpected error for input '%s': %v", test.input, err)
		}

		if header != test.header {
			t.Errorf("Expected header '%s', got '%s' for input: %s", test.header, header, test.input)
		}

		if !slices.Equal(body, test.body) {
			t.Errorf("Expected body '%s', got '%s' for input: %s", test.body, string(body), test.input)
		}
	}
}

func TestGetMimeTypeAndLang(t *testing.T) {
	t.Parallel()
	tests := []struct {
		header   string
		code     int
		mimeType string
		lang     string
	}{
		{"20 text/gemini lang=en", 20, "text/gemini", "en"},
		{"20 text/gemini", 20, "text/gemini", ""},
		{"31 gemini://redirected.to/other/site", 31, "", ""},
		{"20 text/plain;charset=utf-8", 20, "text/plain", ""},
		{"20 text/plain;lang=el-GR", 20, "text/plain", "el-GR"},
		{"20 text/gemini;lang=en-US;charset=utf-8", 20, "text/gemini", "en-US"}, // charset should be ignored
		{"Invalid header", 0, "", ""},
		{"99", 99, "", ""},
	}

	for _, test := range tests {
		code, mimeType, lang := getMimeTypeAndLang(test.header)

		if code != test.code {
			t.Errorf("Expected code %d, got %d for header: %s", test.code, code, test.header)
		}

		if mimeType != test.mimeType {
			t.Errorf("Expected mimeType '%s', got '%s' for header: %s", test.mimeType, mimeType, test.header)
		}

		if lang != test.lang {
			t.Errorf("Expected lang '%s', got '%s' for header: %s", test.lang, lang, test.header)
		}
	}
}

func TestProcessData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		inputData     []byte
		expectedCode  int
		expectedMime  string
		expectedLang  string
		expectedData  []byte
		expectedError bool
	}{
		{
			name:          "Gemini document",
			inputData:     []byte("20 text/gemini\r\n# Hello\nWorld"),
			expectedCode:  20,
			expectedMime:  "text/gemini",
			expectedLang:  "",
			expectedData:  []byte("# Hello\nWorld"),
			expectedError: false,
		},
		{
			name:          "Gemini document with language",
			inputData:     []byte("20 text/gemini lang=en\r\n# Hello\nWorld"),
			expectedCode:  20,
			expectedMime:  "text/gemini",
			expectedLang:  "en",
			expectedData:  []byte("# Hello\nWorld"),
			expectedError: false,
		},
		{
			name:          "Non-Gemini document",
			inputData:     []byte("20 text/html\r\n<h1>Hello</h1>"),
			expectedCode:  20,
			expectedMime:  "text/html",
			expectedLang:  "",
			expectedData:  []byte("<h1>Hello</h1>"),
			expectedError: false,
		},
		{
			name:          "Error header",
			inputData:     []byte("53 No proxying!\r\n"),
			expectedCode:  53,
			expectedMime:  "",
			expectedLang:  "",
			expectedData:  []byte(""),
			expectedError: false,
		},
		{
			name:          "Invalid header",
			inputData:     []byte("Invalid header"),
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := snapshot.Snapshot{}
			result, err := ProcessData(s, test.inputData)

			if test.expectedError && err == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			if !test.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if test.expectedError {
				return
			}

			if int(result.ResponseCode.ValueOrZero()) != test.expectedCode {
				t.Errorf("Expected code %d, got %d", test.expectedCode, int(result.ResponseCode.ValueOrZero()))
			}

			if result.MimeType.ValueOrZero() != test.expectedMime {
				t.Errorf("Expected mimeType '%s', got '%s'", test.expectedMime, result.MimeType.ValueOrZero())
			}

			if result.Lang.ValueOrZero() != test.expectedLang {
				t.Errorf("Expected lang '%s', got '%s'", test.expectedLang, result.Lang.ValueOrZero())
			}

			if test.expectedMime == "text/gemini" {
				if !strings.Contains(result.GemText.String, string(test.expectedData)) {
					t.Errorf("Expected GemText '%s', got '%s'", test.expectedData, result.GemText.String)
				}
			} else {
				if !slices.Equal(result.Data.ValueOrZero(), test.expectedData) {
					t.Errorf("Expected data '%s', got '%s'", test.expectedData, result.Data.ValueOrZero())
				}
			}
		})
	}
}
