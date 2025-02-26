package blackList

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gemini-grc/config"
	"gemini-grc/logging"
	"github.com/antanst/go_errors"
)

var Blacklist []regexp.Regexp //nolint:gochecknoglobals

func LoadBlacklist() error {
	if config.CONFIG.BlacklistPath == "" {
		return nil
	}
	if Blacklist == nil {
		data, err := os.ReadFile(config.CONFIG.BlacklistPath)
		if err != nil {
			Blacklist = []regexp.Regexp{}
			return go_errors.NewError(fmt.Errorf("could not load Blacklist file: %w", err))
		}

		lines := strings.Split(string(data), "\n")

		for _, line := range lines {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			regex, err := regexp.Compile(line)
			if err != nil {
				return go_errors.NewError(fmt.Errorf("could not compile Blacklist line %s: %w", line, err))
			}
			Blacklist = append(Blacklist, *regex)

		}

		if len(lines) > 0 {
			logging.LogInfo("Loaded %d blacklist entries", len(Blacklist))
		}
	}
	return nil
}

func IsBlacklisted(u string) bool {
	for _, v := range Blacklist {
		if v.MatchString(u) {
			return true
		}
	}
	return false
}
