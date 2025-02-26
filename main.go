package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gemini-grc/common"
	"gemini-grc/common/blackList"
	"gemini-grc/config"
	"gemini-grc/db"
	"gemini-grc/errors"
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
	err := runApp()
	if err != nil {
		var asErr *errors.Error
		if errors.As(err, &asErr) {
			logging.LogError("Unexpected error: %v", err)
			_, _ = fmt.Fprintf(os.Stderr, "Unexpected error: %v", err)
		} else {
			logging.LogError("Unexpected error: %v", err)
		}
		os.Exit(1)
	}
}

func runApp() (err error) {
	logging.LogInfo("gemcrawl %s starting up. Press Ctrl+C to exit", common.VERSION)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	_db, err := db.ConnectToDB()
	if err != nil {
		return err
	}

	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(_db)

	err = blackList.LoadBlacklist()
	if err != nil {
		return err
	}

	common.StatusChan = make(chan common.WorkerStatus, config.CONFIG.NumOfWorkers)
	common.ErrorsChan = make(chan error, config.CONFIG.NumOfWorkers)

	// If there's an argument, visit this
	// URL only and don't spawn other workers
	if len(os.Args) > 1 {
		url := os.Args[1]
		err = common.CrawlOneURL(_db, &url)
		return err
	}

	go common.SpawnWorkers(config.CONFIG.NumOfWorkers, _db)

	for {
		select {
		case <-signals:
			logging.LogWarn("Received SIGINT or SIGTERM signal, exiting")
			return nil
		case err := <-common.ErrorsChan:
			if errors.IsFatal(err) {
				return err
			}
			logging.LogError("%s", fmt.Sprintf("%v", err))
		}
	}
}
