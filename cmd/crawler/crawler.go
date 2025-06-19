package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gemini-grc/common"
	"gemini-grc/common/blackList"
	"gemini-grc/common/contextlog"
	"gemini-grc/common/seedList"
	"gemini-grc/common/whiteList"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	gemdb "gemini-grc/db"
	"gemini-grc/robotsMatch"
	"gemini-grc/util"
	"git.antanst.com/antanst/logging"
	"github.com/jmoiron/sqlx"
)

var jobs chan string

func main() {
	var err error

	err = initializeApp()
	if err != nil {
		handleUnexpectedError(err)
	}

	err = runApp()
	if err != nil {
		handleUnexpectedError(err)
	}

	err = shutdownApp()
	if err != nil {
		handleUnexpectedError(err)
	}
}

func handleUnexpectedError(err error) {
	logging.LogError("Unexpected error: %v", err)
	_ = shutdownApp()
	os.Exit(1)
}

func initializeApp() error {
	config.CONFIG = *config.Initialize()
	logging.InitSlogger(config.CONFIG.LogLevel)

	logging.LogInfo("Starting up. Press Ctrl+C to exit")
	common.SignalsChan = make(chan os.Signal, 1)
	signal.Notify(common.SignalsChan, syscall.SIGINT, syscall.SIGTERM)
	common.FatalErrorsChan = make(chan error)
	jobs = make(chan string, config.CONFIG.NumOfWorkers)

	var err error

	err = blackList.Initialize()
	if err != nil {
		return err
	}

	err = whiteList.Initialize()
	if err != nil {
		return err
	}

	err = seedList.Initialize()
	if err != nil {
		return err
	}

	err = robotsMatch.Initialize()
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = gemdb.Database.Initialize(ctx)
	if err != nil {
		return err
	}

	if config.CONFIG.SeedUrlPath != "" {
		err := AddURLsFromFile(ctx, config.CONFIG.SeedUrlPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func shutdownApp() error {
	var err error

	err = blackList.Shutdown()
	if err != nil {
		return err
	}

	err = whiteList.Shutdown()
	if err != nil {
		return err
	}

	err = seedList.Shutdown()
	if err != nil {
		return err
	}

	err = robotsMatch.Shutdown()
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = gemdb.Database.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func runApp() (err error) {
	go spawnWorkers(config.CONFIG.NumOfWorkers)
	go runJobScheduler()
	for {
		select {
		case <-common.SignalsChan:
			logging.LogWarn("Received SIGINT or SIGTERM signal, exiting")
			return nil
		case err := <-common.FatalErrorsChan:
			return err
		}
	}
}

func spawnWorkers(total int) {
	for id := 0; id < total; id++ {
		go func(a int) {
			for {
				job := <-jobs
				common.RunWorkerWithTx(a, job)
			}
		}(id)
	}
}

// Current Logic Flow:
//
// 1. Create transaction
// 2. Get distinct hosts
// 3. If no hosts → fetch snapshots from history (adds URLs to queue)
// 4. Re-query for hosts (should now have some)
// 5. Get URLs from hosts
// 6. Commit transaction
// 7. Queue URLs for workers
func runJobScheduler() {
	var tx *sqlx.Tx
	var err error

	ctx := contextutil.ContextWithComponent(context.Background(), "crawler")
	tx, err = gemdb.Database.NewTx(ctx)
	if err != nil {
		common.FatalErrorsChan <- err
		return
	}

	defer func(tx *sqlx.Tx) {
		if tx != nil {
			if err := gemdb.SafeRollback(ctx, tx); err != nil {
				common.FatalErrorsChan <- err
			}
		}
	}(tx)

	// First, check if the URLs table is empty.
	var urlCount int

	if config.CONFIG.GopherEnable {
		err = tx.Get(&urlCount, "SELECT COUNT(*) FROM urls")
	} else {
		err = tx.Get(&urlCount, "SELECT COUNT(*) FROM urls WHERE url LIKE 'gemini://%'")
	}
	if err != nil {
		common.FatalErrorsChan <- err
		return
	}

	err = tx.Commit()
	if err != nil {
		common.FatalErrorsChan <- err
		return
	}

	// If no pending URLs, add the ones from the standard crawl set.
	tx, err = gemdb.Database.NewTx(ctx)
	if err != nil {
		common.FatalErrorsChan <- err
		return
	}

	if urlCount == 0 {
		logging.LogInfo("URLs table is empty, enqueueing standard crawl set")
		err = enqueueSeedURLs(ctx, tx)
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}
		// Commit this tx here so the loop sees the changes.
		err := tx.Commit()
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}
	} else {
		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "Found %d pending URLs to crawl.", urlCount)
	}

	// Main job loop.
	// We get URLs from the pending URLs table,
	// add crawling jobs for those,
	// and sleep a bit after each run.
	for {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Polling DB for jobs")

		// Use fresh context for DB operations to avoid timeouts/cancellation
		// from the long-lived scheduler context affecting database transactions
		dbCtx := context.Background()
		tx, err = gemdb.Database.NewTx(dbCtx)
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}

		// Get all distinct hosts from pending URLs
		distinctHosts, err := gemdb.Database.GetUrlHosts(dbCtx, tx)
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}

		// When out of pending URLs, add some random ones.
		if len(distinctHosts) == 0 {
			// Queue random old URLs from history.
			count, err := fetchSnapshotsFromHistory(dbCtx, tx, config.CONFIG.NumOfWorkers*10, config.CONFIG.SkipIfUpdatedDays)
			if err != nil {
				common.FatalErrorsChan <- err
				return
			}
			if count == 0 {
				contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "No work, waiting to poll DB...")
				time.Sleep(120 * time.Second)
				continue
			}
			distinctHosts, err = gemdb.Database.GetUrlHosts(dbCtx, tx)
			if err != nil {
				common.FatalErrorsChan <- err
				return
			}
		}

		// Get some URLs from each host, up to a limit
		urls, err := gemdb.Database.GetRandomUrlsFromHosts(dbCtx, distinctHosts, 10, tx)
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}

		err = tx.Commit()
		if err != nil {
			common.FatalErrorsChan <- err
			return
		}

		if len(urls) == 0 {
			contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "No work, waiting to poll DB...")
			time.Sleep(120 * time.Second)
			continue
		}

		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "Queueing %d distinct hosts -> %d urls to crawl", len(distinctHosts), len(urls))

		// Add jobs to WaitGroup before queuing
		common.WorkerWG.Add(len(urls))

		for _, url := range urls {
			jobs <- url
		}

		// Wait for all workers to complete their jobs
		common.WorkerWG.Wait()

		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "All workers done. New scheduler run starts")
		logging.LogInfo("")
		logging.LogInfo("")
	}
}

func enqueueSeedURLs(ctx context.Context, tx *sqlx.Tx) error {
	// Get seed URLs from seedList module
	//urls := seedList.GetSeedURLs()
	//
	//for _, url := range urls {
	//	err := gemdb.Database.InsertURL(ctx, tx, url)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}

func fetchSnapshotsFromHistory(ctx context.Context, tx *sqlx.Tx, num int, age int) (int, error) {
	// Select <num> snapshots from snapshots table for recrawling
	// Find URLs where the LATEST crawl attempt (via last_crawled) is at least <age> days old
	// Uses last_crawled timestamp to track actual crawl attempts regardless of content changes
	historyCtx := contextutil.ContextWithComponent(context.Background(), "fetchSnapshotsFromHistory")
	contextlog.LogDebugWithContext(historyCtx, logging.GetSlogger(), "Looking for %d URLs whose latest crawl attempt is at least %d days old to recrawl", num, age)

	// Calculate the cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -age)

	// Use the query from db_queries.go to find URLs that need re-crawling

	type SnapshotURL struct {
		URL  string `db:"url"`
		Host string `db:"host"`
	}

	// Execute the query
	var snapshotURLs []SnapshotURL
	err := tx.Select(&snapshotURLs, gemdb.SQL_FETCH_SNAPSHOTS_FROM_HISTORY, cutoffDate, num)
	if err != nil {
		return 0, err
	}

	if len(snapshotURLs) == 0 {
		return 0, nil
	}

	// For each selected snapshot, add the URL to the urls table
	insertCount := 0
	for _, snapshot := range snapshotURLs {
		err := gemdb.Database.InsertURL(ctx, tx, snapshot.URL)
		if err != nil {
			logging.LogError("Error inserting URL %s from old snapshot to queue: %v", snapshot.URL, err)
			return 0, err
		}
		insertCount++
	}

	// Note: The transaction is committed by the caller (runJobScheduler),
	// not here. This function is called as part of a larger transaction.
	if insertCount > 0 {
		contextlog.LogInfoWithContext(historyCtx, logging.GetSlogger(), "Added %d old URLs to recrawl queue", insertCount)
	}

	return insertCount, nil
}

func AddURLsFromFile(ctx context.Context, filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	urls := util.Filter(lines, func(url string) bool {
		return strings.TrimSpace(url) != ""
	})

	// Create a context for database operations
	tx, err := gemdb.Database.NewTx(ctx)
	if err != nil {
		return err
	}

	// Insert all the URLs
	for _, url := range urls {
		fileCtx := contextutil.ContextWithComponent(context.Background(), "AddURLsFromFile")
		contextlog.LogInfoWithContext(fileCtx, logging.GetSlogger(), "Adding %s to queue", url)
		err := gemdb.Database.InsertURL(ctx, tx, url)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
