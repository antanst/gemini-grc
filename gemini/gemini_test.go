package gemini

import (
	"gemini-grc/common"
	"testing"
)

func TestExtractRedirectTargetFullURL(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://smol.gr", "")
	input := "redirect: 31 gemini://target.gr"
	result, err := extractRedirectTarget(*currentURL, input)
	expected := "gemini://target.gr:1965"
	if err != nil || (result.String() != expected) {
		t.Errorf("fail: Expected %s got %s", expected, result)
	}
}

func TestExtractRedirectTargetFullURLSlash(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://smol.gr", "")
	input := "redirect: 31 gemini://target.gr/"
	result, err := extractRedirectTarget(*currentURL, input)
	expected := "gemini://target.gr:1965/"
	if err != nil || (result.String() != expected) {
		t.Errorf("fail: Expected %s got %s", expected, result)
	}
}

func TestExtractRedirectTargetRelativeURL(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://smol.gr", "")
	input := "redirect: 31 /a/b"
	result, err := extractRedirectTarget(*currentURL, input)
	if err != nil || (result.String() != "gemini://smol.gr:1965/a/b") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetRelativeURL2(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://nox.im:1965", "")
	input := "redirect: 31 ./"
	result, err := extractRedirectTarget(*currentURL, input)
	if err != nil || (result.String() != "gemini://nox.im:1965/") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetRelativeURL3(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://status.zvava.org:1965", "")
	input := "redirect: 31 index.gmi"
	result, err := extractRedirectTarget(*currentURL, input)
	if err != nil || (result.String() != "gemini://status.zvava.org:1965/index.gmi") {
		t.Errorf("fail: %s", result)
	}
}

func TestExtractRedirectTargetWrong(t *testing.T) {
	t.Parallel()
	currentURL, _ := common.ParseURL("gemini://smol.gr", "")
	input := "redirect: 31"
	result, err := extractRedirectTarget(*currentURL, input)
	if result != nil || err == nil {
		t.Errorf("fail: result should be nil, err is %s", err)
	}
}
