package common_test

import (
	"reflect"
	"testing"

	"gemini-grc/common"
)

func TestParseURL(t *testing.T) {
	t.Parallel()
	input := "gemini://caolan.uk/cgi-bin/weather.py/wxfcs/3162"
	parsed, err := common.ParseURL(input, "", true)
	value, _ := parsed.Value()
	if err != nil || !(value == "gemini://caolan.uk:1965/cgi-bin/weather.py/wxfcs/3162") {
		t.Errorf("fail: %s", parsed)
	}
}

func TestDeriveAbsoluteURL_abs_url_input(t *testing.T) {
	t.Parallel()
	currentURL := common.URL{
		Protocol: "gemini",
		Hostname: "smol.gr",
		Port:     1965,
		Path:     "/a/b",
		Descr:    "Nothing",
		Full:     "gemini://smol.gr:1965/a/b",
	}
	input := "gemini://a.b/c"
	output, err := common.DeriveAbsoluteURL(currentURL, input)
	if err != nil {
		t.Errorf("fail: %v", err)
	}
	expected := &common.URL{
		Protocol: "gemini",
		Hostname: "a.b",
		Port:     1965,
		Path:     "/c",
		Descr:    "",
		Full:     "gemini://a.b:1965/c",
	}
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestDeriveAbsoluteURL_abs_path_input(t *testing.T) {
	t.Parallel()
	currentURL := common.URL{
		Protocol: "gemini",
		Hostname: "smol.gr",
		Port:     1965,
		Path:     "/a/b",
		Descr:    "Nothing",
		Full:     "gemini://smol.gr:1965/a/b",
	}
	input := "/c"
	output, err := common.DeriveAbsoluteURL(currentURL, input)
	if err != nil {
		t.Errorf("fail: %v", err)
	}
	expected := &common.URL{
		Protocol: "gemini",
		Hostname: "smol.gr",
		Port:     1965,
		Path:     "/c",
		Descr:    "",
		Full:     "gemini://smol.gr:1965/c",
	}
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestDeriveAbsoluteURL_rel_path_input(t *testing.T) {
	t.Parallel()
	currentURL := common.URL{
		Protocol: "gemini",
		Hostname: "smol.gr",
		Port:     1965,
		Path:     "/a/b",
		Descr:    "Nothing",
		Full:     "gemini://smol.gr:1965/a/b",
	}
	input := "c/d"
	output, err := common.DeriveAbsoluteURL(currentURL, input)
	if err != nil {
		t.Errorf("fail: %v", err)
	}
	expected := &common.URL{
		Protocol: "gemini",
		Hostname: "smol.gr",
		Port:     1965,
		Path:     "/a/b/c/d",
		Descr:    "",
		Full:     "gemini://smol.gr:1965/a/b/c/d",
	}
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeURLSlash(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/retro-computing/magazines/"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := input
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeURLNoSlash(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/retro-computing/magazines"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := input
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeMultiSlash(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/retro-computing/////////a///magazines"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/retro-computing/a/magazines"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeTrailingSlash(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeNoTrailingSlash(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeTrailingSlashPath(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/a/"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/a/"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeNoTrailingSlashPath(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/a"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/a"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeDot(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net/retro-computing/./././////a///magazines"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/retro-computing/a/magazines"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizePort(t *testing.T) {
	t.Parallel()
	input := "gemini://uscoffings.net:1965/a"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://uscoffings.net/a"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
	input := "gemini://chat.gemini.lehmann.cx:11965/"
	normalized, _ := common.NormalizeURL(input)
	output := normalized.String()
	expected := "gemini://chat.gemini.lehmann.cx:11965/"
	pass := reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}

	input = "gemini://chat.gemini.lehmann.cx:11965/index?a=1&b=c"
	normalized, _ = common.NormalizeURL(input)
	output = normalized.String()
	expected = "gemini://chat.gemini.lehmann.cx:11965/index?a=1&b=c"
	pass = reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}

	input = "gemini://chat.gemini.lehmann.cx:11965/index#1"
	normalized, _ = common.NormalizeURL(input)
	output = normalized.String()
	expected = "gemini://chat.gemini.lehmann.cx:11965/index#1"
	pass = reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}

	input = "gemini://gemi.dev/cgi-bin/xkcd.cgi?1494"
	normalized, _ = common.NormalizeURL(input)
	output = normalized.String()
	expected = "gemini://gemi.dev/cgi-bin/xkcd.cgi?1494"
	pass = reflect.DeepEqual(output, expected)
	if !pass {
		t.Errorf("fail: %#v != %#v", output, expected)
	}
}
