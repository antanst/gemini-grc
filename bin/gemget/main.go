package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gemini-grc/common/snapshot"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/gemini"
	"gemini-grc/gopher"
	"gemini-grc/logging"
	"github.com/antanst/go_errors"
)

func main() {
	config.CONFIG = *config.GetConfig()
	err := runApp()
	if err != nil {
		fmt.Printf("%v\n", err)
		logging.LogError("%v", err)
		os.Exit(1)
	}
}

func runApp() error {
	if len(os.Args) != 2 {
		return go_errors.NewError(fmt.Errorf("missing URL to visit"))
	}
	url := os.Args[1]
	var s *snapshot.Snapshot
	var err error
	if _url.IsGeminiUrl(url) {
		s, err = gemini.Visit(url)
	} else if _url.IsGopherURL(url) {
		s, err = gopher.Visit(url)
	} else {
		return go_errors.NewFatalError(fmt.Errorf("not a Gemini or Gopher URL"))
	}
	if err != nil {
		return err
	}
	_json, _ := json.MarshalIndent(s, "", "  ")
	fmt.Printf("%s\n", _json)
	return err
}
