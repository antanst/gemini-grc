package whiteList

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gemini-grc/config"
	"gemini-grc/logging"
	"git.antanst.com/antanst/xerrors"
)

var whitelist []regexp.Regexp //nolint:gochecknoglobals

func Initialize() error {
	var err error

	// Initialize whitelist
	if config.CONFIG.WhitelistPath != "" {
		if err = loadWhitelist(config.CONFIG.WhitelistPath); err != nil {
			return err
		}
	}

	return nil
}

func loadWhitelist(filePath string) error {
	if whitelist != nil {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		whitelist = []regexp.Regexp{}
		return xerrors.NewError(fmt.Errorf("could not load whitelist file: %w", err), 0, "", true)
	}

	lines := strings.Split(string(data), "\n")
	whitelist = []regexp.Regexp{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		regex, err := regexp.Compile(line)
		if err != nil {
			return xerrors.NewError(fmt.Errorf("could not compile whitelist line %s: %w", line, err), 0, "", true)
		}
		whitelist = append(whitelist, *regex)
	}

	if len(whitelist) > 0 {
		logging.LogInfo("Loaded %d whitelist entries", len(whitelist))
	}

	return nil
}

func Shutdown() error {
	return nil
}

// IsWhitelisted checks if the URL matches any whitelist pattern
func IsWhitelisted(u string) bool {
	for _, v := range whitelist {
		if v.MatchString(u) {
			return true
		}
	}
	return false
}
