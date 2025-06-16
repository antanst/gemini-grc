package seedList

import (
	"fmt"
	"os"
	"strings"

	"git.antanst.com/antanst/logging"
	"git.antanst.com/antanst/xerrors"
)

var seedlist []string //nolint:gochecknoglobals

func Initialize() error {
	var err error

	// Initialize seedlist from fixed path
	if err = loadSeedlist("seed_urls.txt"); err != nil {
		return err
	}

	return nil
}

func loadSeedlist(filePath string) error {
	if seedlist != nil {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		seedlist = []string{}
		return xerrors.NewError(fmt.Errorf("could not load seedlist file: %w", err), 0, "", true)
	}

	lines := strings.Split(string(data), "\n")
	seedlist = []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		seedlist = append(seedlist, line)
	}

	if len(seedlist) > 0 {
		logging.LogInfo("Loaded %d seed URLs", len(seedlist))
	}

	return nil
}

func Shutdown() error {
	return nil
}

// GetSeedURLs returns the list of seed URLs
func GetSeedURLs() []string {
	if seedlist == nil {
		return []string{}
	}
	// Return a copy to prevent external modification
	result := make([]string, len(seedlist))
	copy(result, seedlist)
	return result
}
