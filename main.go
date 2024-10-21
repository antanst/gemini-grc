package main

import (
	"gemini-grc/config"
	"gemini-grc/gemini"
	"gemini-grc/logging"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	config.CONFIG = *config.GetConfig()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(config.CONFIG.LogLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "[2006-01-02 15:04:05]"})
	if err := runApp(); err != nil {
		logging.LogError("Application error: %w", err)
		os.Exit(1)
	}
}

func runApp() error {
	logging.LogInfo("Starting up. Press Ctrl+C to exit")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	db := gemini.ConnectToDB()

	// !!! DANGER !!!
	// Removes all rows and adds some seed URLs.
	// populateDB(db)

	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			// TODO properly log & hangle error
			panic(err)
		}
	}(db)

	go gemini.SpawnWorkers(config.CONFIG.NumOfWorkers, db)

	<-sigs
	logging.LogInfo("Received SIGINT or SIGTERM signal, exiting")
	return nil
}
