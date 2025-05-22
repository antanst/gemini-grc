package blackList

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gemini-grc/config"
	"gemini-grc/logging"
	"git.antanst.com/antanst/xerrors"
)

var blacklist []regexp.Regexp //nolint:gochecknoglobals

func Initialize() error {
	var err error

	// Initialize blacklist
	if config.CONFIG.BlacklistPath != "" {
		if err = loadBlacklist(config.CONFIG.BlacklistPath); err != nil {
			return err
		}
	}

	return nil
}

func loadBlacklist(filePath string) error {
	if blacklist != nil {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		blacklist = []regexp.Regexp{}
		return xerrors.NewError(fmt.Errorf("could not load blacklist file: %w", err), 0, "", true)
	}

	lines := strings.Split(string(data), "\n")
	blacklist = []regexp.Regexp{}

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		regex, err := regexp.Compile(line)
		if err != nil {
			return xerrors.NewError(fmt.Errorf("could not compile blacklist line %s: %w", line, err), 0, "", true)
		}
		blacklist = append(blacklist, *regex)
	}

	if len(blacklist) > 0 {
		logging.LogInfo("Loaded %d blacklist entries", len(blacklist))
	}

	return nil
}

func Shutdown() error {
	return nil
}

// IsBlacklisted checks if the URL matches any blacklist pattern
func IsBlacklisted(u string) bool {
	for _, v := range blacklist {
		if v.MatchString(u) {
			return true
		}
	}
	return false
}
