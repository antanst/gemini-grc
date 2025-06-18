package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	PgURL             string
	LogLevel          slog.Level // Logging level (debug, info, warn, error)
	MaxResponseSize   int        // Maximum size of response in bytes
	MaxDbConnections  int        // Maximum number of database connections.
	NumOfWorkers      int        // Number of concurrent workers
	ResponseTimeout   int        // Timeout for responses in seconds
	BlacklistPath     string     // File that has blacklisted strings of "host:port"
	WhitelistPath     string     // File with URLs that should always be crawled regardless of blacklist
	DryRun            bool       // If false, don't write to disk
	GopherEnable      bool       // Enable Gopher crawling
	SeedUrlPath       string     // Add URLs from file to queue
	SkipIfUpdatedDays int        // Skip re-crawling URLs updated within this many days (0 to disable)
}

var CONFIG Config //nolint:gochecknoglobals

// Initialize loads and validates configuration from environment variables
func Initialize() *Config {
	config := &Config{}

	loglevel := flag.String("log-level", "info", "Logging level (debug, info, warn, error)")
	pgURL := flag.String("pgurl", "", "Postgres URL")
	dryRun := flag.Bool("dry-run", false, "Dry run mode")
	gopherEnable := flag.Bool("gopher", false, "Enable crawling of Gopher holes")
	maxDbConnections := flag.Int("max-db-connections", 100, "Maximum number of database connections")
	numOfWorkers := flag.Int("workers", 1, "Number of concurrent workers")
	maxResponseSize := flag.Int("max-response-size", 1024*1024, "Maximum size of response in bytes")
	responseTimeout := flag.Int("response-timeout", 10, "Timeout for network responses in seconds")
	blacklistPath := flag.String("blacklist-path", "", "File that has blacklist regexes")
	skipIfUpdatedDays := flag.Int("skip-if-updated-days", 60, "Skip re-crawling URLs updated within this many days (0 to disable)")
	whitelistPath := flag.String("whitelist-path", "", "File with URLs that should always be crawled regardless of blacklist")
	seedUrlPath := flag.String("seed-url-path", "", "File with seed URLs that should be added to the queue immediatelly")

	flag.Parse()

	config.PgURL = *pgURL
	config.DryRun = *dryRun
	config.GopherEnable = *gopherEnable
	config.NumOfWorkers = *numOfWorkers
	config.MaxResponseSize = *maxResponseSize
	config.ResponseTimeout = *responseTimeout
	config.BlacklistPath = *blacklistPath
	config.WhitelistPath = *whitelistPath
	config.SeedUrlPath = *seedUrlPath
	config.MaxDbConnections = *maxDbConnections
	config.SkipIfUpdatedDays = *skipIfUpdatedDays

	level, err := ParseSlogLevel(*loglevel)
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		os.Exit(-1)
	}
	config.LogLevel = level

	return config
}

// ParseSlogLevel converts a string level to slog.Level
func ParseSlogLevel(levelStr string) (slog.Level, error) {
	switch levelStr {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", levelStr)
	}
}

// Convert method for backward compatibility with existing codebase
// This can be removed once all references to Convert() are updated
func (c *Config) Convert() *Config {
	// Just return the config itself as it now directly contains slog.Level
	return c
}
