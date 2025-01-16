package gemini

import (
	"errors"
	"reflect"
	"testing"

	"gemini-grc/common"
)

type TestData struct {
	currentURL string
	link       string
	value      *common.URL
	error      error
}

var data = []TestData{
	{
		currentURL: "https://gemini.com/",
		link:       "https://gemini.com/",
		value:      nil,
		error:      common.ErrGeminiLinkLineParse,
	},
	{
		currentURL: "gemini://gemi.dev/cgi-bin/xkcd/",
		link:       "=> archive/ Complete Archive",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "gemi.dev",
			Port:     1965,
			Path:     "/cgi-bin/xkcd/archive/",
			Descr:    "Complete Archive",
			Full:     "gemini://gemi.dev:1965/cgi-bin/xkcd/archive/",
		},
		error: nil,
	},
	{
		currentURL: "gemini://gemi.dev/cgi-bin/xkcd/",
		link:       "=> /cgi-bin/xkcd.cgi?a=5&b=6 Example",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "gemi.dev",
			Port:     1965,
			Path:     "/cgi-bin/xkcd.cgi",
			Descr:    "Example",
			Full:     "gemini://gemi.dev:1965/cgi-bin/xkcd.cgi?a=5&b=6",
		},
		error: nil,
	},
	{
		currentURL: "gemini://gemi.dev/cgi-bin/xkcd/",
		link:       "=> /cgi-bin/xkcd.cgi?1494 XKCD 1494: Insurance",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "gemi.dev",
			Port:     1965,
			Path:     "/cgi-bin/xkcd.cgi",
			Descr:    "XKCD 1494: Insurance",
			Full:     "gemini://gemi.dev:1965/cgi-bin/xkcd.cgi?1494",
		},
		error: nil,
	},
	{
		currentURL: "gemini://gemi.dev/cgi-bin/xkcd/",
		link:       "=> /cgi-bin/xkcd.cgi?1494#f XKCD 1494: Insurance",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "gemi.dev",
			Port:     1965,
			Path:     "/cgi-bin/xkcd.cgi",
			Descr:    "XKCD 1494: Insurance",
			Full:     "gemini://gemi.dev:1965/cgi-bin/xkcd.cgi?1494#f",
		},
		error: nil,
	},
	{
		currentURL: "gemini://gemi.dev/cgi-bin/xkcd/",
		link:       "=> /cgi-bin/xkcd.cgi?c=5#d XKCD 1494: Insurance",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "gemi.dev",
			Port:     1965,
			Path:     "/cgi-bin/xkcd.cgi",
			Descr:    "XKCD 1494: Insurance",
			Full:     "gemini://gemi.dev:1965/cgi-bin/xkcd.cgi?c=5#d",
		},
		error: nil,
	},
	{
		currentURL: "gemini://a.b/c#d",
		link:       "=> /d/e#f",
		value: &common.URL{
			Protocol: "gemini",
			Hostname: "a.b",
			Port:     1965,
			Path:     "/d/e",
			Descr:    "",
			Full:     "gemini://a.b:1965/d/e#f",
		},
		error: nil,
	},
}

func Test(t *testing.T) {
	t.Parallel()
	for i, expected := range data {
		result, err := ParseGeminiLinkLine(expected.link, expected.currentURL)
		if err != nil { //nolint:nestif
			if expected.value != nil {
				t.Errorf("data[%d]: Expected value %v, got %v", i, nil, expected.value)
			}
			if !errors.Is(err, common.ErrGeminiLinkLineParse) {
				t.Errorf("data[%d]: expected error %v, got %v", i, expected.error, err)
			}
		} else {
			if expected.error != nil {
				t.Errorf("data[%d]: Expected error %v, got %v", i, nil, expected.error)
			}
			if !(reflect.DeepEqual(result, expected.value)) {
				t.Errorf("data[%d]: expected %#v, got %#v", i, expected.value, result)
			}
		}
	}
}
