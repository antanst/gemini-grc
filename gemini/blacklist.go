package gemini

import (
	"fmt"
	"gemini-grc/common"
	"os"
	"strings"

	"gemini-grc/config"
	"gemini-grc/logging"
)

var Blacklist *[]string //nolint:gochecknoglobals

func LoadBlacklist() {
	if Blacklist == nil {
		data, err := os.ReadFile(config.CONFIG.BlacklistPath)
		if err != nil {
			Blacklist = &[]string{}
			logging.LogWarn("Could not load Blacklist file: %v", err)
			return
		}
		lines := strings.Split(string(data), "\n")

		// Ignore lines starting with '#' (comments)
		filteredLines := func() []string {
			out := make([]string, 0, len(lines))
			for _, line := range lines {
				if !strings.HasPrefix(line, "#") {
					out = append(out, line)
				}
			}
			return out
		}()

		if len(lines) > 0 {
			Blacklist = &filteredLines
			logging.LogInfo("Blacklist has %d entries", len(*Blacklist))
		}
	}
}

func IsBlacklisted(u string) bool {
	url, err := common.ParseURL(u, "")
	if err != nil {
		return false
	}
	hostWithPort := fmt.Sprintf("%s:%d", url.Hostname, url.Port)
	for _, v := range *Blacklist {
		if v == url.Hostname || v == hostWithPort {
			return true
		}
	}
	return false
}
