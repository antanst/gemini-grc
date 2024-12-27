package main

import (
	"os"
	"os/signal"
	"syscall"

	"gemini-grc/config"
	"gemini-grc/gemini"
	"gemini-grc/logging"
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
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

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

	gemini.LoadBlacklist()

	// If there's an argument, visit this
	// URL only and don't spawn other workers
	if len(os.Args) > 1 {
		url := os.Args[1]
		go gemini.RunWorkerWithTx(0, db, &url)
	} else {
		go gemini.SpawnWorkers(config.CONFIG.NumOfWorkers, db)
	}

	<-signals
	logging.LogWarn("Received SIGINT or SIGTERM signal, exiting")
	return nil
}
