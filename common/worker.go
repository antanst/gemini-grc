package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gemini-grc/common/blackList"
	"gemini-grc/common/contextlog"
	commonErrors "gemini-grc/common/errors"
	"gemini-grc/common/snapshot"
	url2 "gemini-grc/common/url"
	"gemini-grc/common/whiteList"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	gemdb "gemini-grc/db"
	"gemini-grc/gemini"
	"gemini-grc/gopher"
	"gemini-grc/hostPool"
	"gemini-grc/logging"
	"gemini-grc/robotsMatch"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

func RunWorkerWithTx(workerID int, job string) {
	// Extract host from URL for the context.
	parsedURL, err := url2.ParseURL(job, "", true)
	if err != nil {
		logging.LogInfo("Failed to parse job URL: %s Error: %s", job, err)
		return
	}
	host := parsedURL.Hostname

	// Create a new worker context
	baseCtx := context.Background()
	ctx, cancel := contextutil.NewRequestContext(baseCtx, job, host, workerID)
	defer cancel() // Ensure the context is cancelled when we're done
	ctx = contextutil.ContextWithComponent(ctx, "worker")
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Starting worker for URL %s", job)

	// Create a new db transaction
	tx, err := gemdb.Database.NewTx(ctx)
	if err != nil {
		FatalErrorsChan <- err
		return
	}

	err = runWorker(ctx, tx, []string{job})
	if err != nil {
		// Handle context cancellation and timeout errors gracefully, instead of treating them as fatal
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Worker timed out or canceled: %v", err)
			rollbackErr := SafeRollback(ctx, tx)
			if rollbackErr != nil {
				FatalErrorsChan <- rollbackErr
				return
			}
			return
		}
		// For other errors, we treat them as fatal.
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Worker failed: %v", err)
		rollbackErr := SafeRollback(ctx, tx)
		if rollbackErr != nil {
			FatalErrorsChan <- rollbackErr
		}
		FatalErrorsChan <- err
		return
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Committing transaction")
	err = tx.Commit()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to commit transaction: %v", err)
		if rollbackErr := SafeRollback(ctx, tx); rollbackErr != nil {
			FatalErrorsChan <- err
			return
		}
	}
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Worker done!")
}

// SafeRollback attempts to roll back a transaction,
// handling the case if the tx was already finalized.
func SafeRollback(ctx context.Context, tx *sqlx.Tx) error {
	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		// Check if it's the standard "transaction already finalized" error
		if errors.Is(rollbackErr, sql.ErrTxDone) {
			contextlog.LogWarnWithContext(ctx, logging.GetSlogger(), "Rollback failed because transaction is already finalized")
			return nil
		}
		// Only panic for other types of rollback failures
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to rollback transaction: %v", rollbackErr)
		return xerrors.NewError(fmt.Errorf("failed to rollback transaction: %w", rollbackErr), 0, "", true)
	}
	return nil
}

func runWorker(ctx context.Context, tx *sqlx.Tx, urls []string) error {
	total := len(urls)
	for i, u := range urls {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Starting %d/%d %s", i+1, total, u)
		urlCtx, cancelFunc := context.WithCancel(ctx)
		err := WorkOnUrl(urlCtx, tx, u)
		cancelFunc()
		if err != nil {
			return err
		}
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Done %d/%d.", i+1, total)
	}
	return nil
}

// WorkOnUrl visits a URL and stores the result.
// unexpected errors are returned.
// expected errors are stored within the snapshot.
func WorkOnUrl(ctx context.Context, tx *sqlx.Tx, url string) (err error) {
	// Create a context specifically for this URL with "url" component
	urlCtx := contextutil.ContextWithComponent(ctx, "url")

	contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Processing URL: %s", url)

	s, err := snapshot.SnapshotFromURL(url, true)
	if err != nil {
		contextlog.LogErrorWithContext(urlCtx, logging.GetSlogger(), "Failed to parse URL: %v", err)
		return err
	}

	isGemini := url2.IsGeminiUrl(s.URL.String())
	isGopher := url2.IsGopherURL(s.URL.String())

	if !isGemini && !isGopher {
		return xerrors.NewError(fmt.Errorf("not a Gopher or Gemini URL: %s", s.URL.String()), 0, "", false)
	}

	if isGopher && !config.CONFIG.GopherEnable {
		contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Skipping gopher URL (disabled in config)")
		return nil
	}

	if url != s.URL.Full {
		err = gemdb.Database.NormalizeURL(ctx, tx, url, s.URL.Full)
		if err != nil {
			return err
		}
		contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Normalized URL: %s → %s", url, s.URL.Full)
		url = s.URL.Full
	}

	// Check if URL is whitelisted
	isUrlWhitelisted := whiteList.IsWhitelisted(s.URL.String())
	if isUrlWhitelisted {
		contextlog.LogInfoWithContext(urlCtx, logging.GetSlogger(), "URL matches whitelist, forcing crawl %s", url)
	}

	// Only check blacklist if URL is not whitelisted
	if !isUrlWhitelisted && blackList.IsBlacklisted(s.URL.String()) {
		contextlog.LogInfoWithContext(urlCtx, logging.GetSlogger(), "URL matches blacklist, ignoring %s", url)
		s.Error = null.StringFrom(commonErrors.ErrBlacklistMatch.Error())
		return saveSnapshotAndRemoveURL(ctx, tx, s)
	}

	// Only check robots.txt if URL is not whitelisted and is a Gemini URL
	var robotMatch bool
	if !isUrlWhitelisted && isGemini {
		// If URL matches a robots.txt disallow line,
		// add it as an error and remove url
		robotMatch, err = robotsMatch.RobotMatch(urlCtx, s.URL.String())
		if err != nil {
			if commonErrors.IsHostError(err) {
				return removeURL(ctx, tx, s.URL.String())
			}
			return err
		}
		if robotMatch {
			contextlog.LogInfoWithContext(urlCtx, logging.GetSlogger(), "URL matches robots.txt, skipping")
			s.Error = null.StringFrom(commonErrors.ErrRobotsMatch.Error())
			return saveSnapshotAndRemoveURL(ctx, tx, s)
		}
	}

	contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Adding to host pool")
	err = hostPool.AddHostToHostPool(urlCtx, s.Host)
	if err != nil {
		contextlog.LogErrorWithContext(urlCtx, logging.GetSlogger(), "Failed to add host to pool: %v", err)
		return err
	}

	defer func(ctx context.Context, host string) {
		hostPool.RemoveHostFromPool(ctx, host)
	}(urlCtx, s.Host)

	contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Visiting %s", s.URL.String())

	// Use context-aware visits for both protocols
	if isGopher {
		// Use the context-aware version for Gopher visits
		s, err = gopher.VisitWithContext(urlCtx, s.URL.String())
	} else {
		// Use the context-aware version for Gemini visits
		s, err = gemini.Visit(urlCtx, s.URL.String())
	}

	if err != nil {
		contextlog.LogErrorWithContext(urlCtx, logging.GetSlogger(), "Error visiting URL: %v", err)
		return err
	}
	if s == nil {
		contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "No snapshot returned")
		return nil
	}

	// Handle Gemini redirection.
	if isGemini &&
		s.ResponseCode.ValueOrZero() >= 30 &&
		s.ResponseCode.ValueOrZero() < 40 {
		err = handleRedirection(urlCtx, tx, s)
		if err != nil {
			return fmt.Errorf("error while handling redirection: %s", err)
		}
	}

	// Check if content is identical to previous snapshot and we should skip further processing
	if config.CONFIG.SkipIdenticalContent {
		identical, err := gemdb.Database.IsContentIdentical(ctx, tx, s)
		if err != nil {
			return err
		}
		if identical {
			contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Content identical to existing snapshot, skipping")
			return removeURL(ctx, tx, s.URL.String())
		}
	}

	// Process and store links since content has changed
	if len(s.Links.ValueOrZero()) > 0 {
		contextlog.LogDebugWithContext(urlCtx, logging.GetSlogger(), "Found %d links", len(s.Links.ValueOrZero()))
		err = storeLinks(ctx, tx, s)
		if err != nil {
			return err
		}
	}

	// Save the snapshot and remove the URL from the queue
	contextlog.LogInfoWithContext(urlCtx, logging.GetSlogger(), "%2d %s", s.ResponseCode.ValueOrZero(), s.URL.String())
	return saveSnapshotAndRemoveURL(ctx, tx, s)
}

// storeLinks checks and stores the snapshot links in the database.
func storeLinks(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	if s.Links.Valid { //nolint:nestif
		for _, link := range s.Links.ValueOrZero() {
			if shouldPersistURL(&link) {
				visited, err := haveWeVisitedURL(ctx, tx, link.Full)
				if err != nil {
					return err
				}
				if !visited {
					err := gemdb.Database.InsertURL(ctx, tx, link.Full)
					if err != nil {
						return err
					}
				} else {
					contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Link already persisted: %s", link.Full)
				}
			}
		}
	}
	return nil
}

// Context-aware version of removeURL
func removeURL(ctx context.Context, tx *sqlx.Tx, url string) error {
	return gemdb.Database.DeleteURL(ctx, tx, url)
}

// Context-aware version of saveSnapshotAndRemoveURL
func saveSnapshotAndRemoveURL(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	err := gemdb.Database.SaveSnapshot(ctx, tx, s)
	if err != nil {
		return err
	}
	return gemdb.Database.DeleteURL(ctx, tx, s.URL.String())
}

// shouldPersistURL returns true given URL is a
// non-blacklisted Gemini or Gopher URL.
func shouldPersistURL(u *url2.URL) bool {
	if blackList.IsBlacklisted(u.String()) {
		return false
	}
	if config.CONFIG.GopherEnable && url2.IsGopherURL(u.String()) {
		return true
	}
	return url2.IsGeminiUrl(u.String())
}

func haveWeVisitedURL(ctx context.Context, tx *sqlx.Tx, u string) (bool, error) {
	var result []bool

	// Check if the context is cancelled
	if err := ctx.Err(); err != nil {
		return false, err
	}

	// Check the urls table which holds the crawl queue.
	err := tx.SelectContext(ctx, &result, `SELECT TRUE FROM urls WHERE url=$1`, u)
	if err != nil {
		return false, xerrors.NewError(fmt.Errorf("database error: %w", err), 0, "", true)
	}
	if len(result) > 0 {
		return false, nil
	}

	// If we're skipping URLs based on recent updates, check if this URL has been
	// crawled within the specified number of days
	if config.CONFIG.SkipIfUpdatedDays > 0 {
		var recentSnapshots []bool
		cutoffDate := time.Now().AddDate(0, 0, -config.CONFIG.SkipIfUpdatedDays)

		// Check if the context is cancelled
		if err := ctx.Err(); err != nil {
			return false, err
		}

		err = tx.SelectContext(ctx, &recentSnapshots, `
			SELECT TRUE FROM snapshots
			WHERE snapshots.url=$1
			AND timestamp > $2
			LIMIT 1`, u, cutoffDate)
		if err != nil {
			return false, xerrors.NewError(fmt.Errorf("database error checking recent snapshots: %w", err), 0, "", true)
		}

		if len(recentSnapshots) > 0 {
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Skipping URL %s (updated within last %d days)", u, config.CONFIG.SkipIfUpdatedDays)
			return true, nil
		}
	}

	return false, nil
}

func handleRedirection(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	// Create a context specifically for redirection handling
	redirectCtx := contextutil.ContextWithComponent(ctx, "redirect")

	// Use the redirectCtx for all operations
	newURL, err := url2.ExtractRedirectTargetFromHeader(s.URL, s.Header.ValueOrZero())
	if err != nil {
		contextlog.LogErrorWithContext(redirectCtx, logging.GetSlogger(), "Failed to extract redirect target: %v", err)
		return err
	}
	contextlog.LogDebugWithContext(redirectCtx, logging.GetSlogger(), "Page redirects to %s", newURL)

	haveWeVisited, err := haveWeVisitedURL(redirectCtx, tx, newURL.String())
	if err != nil {
		return err
	}

	if shouldPersistURL(newURL) && !haveWeVisited {
		err = gemdb.Database.InsertURL(redirectCtx, tx, newURL.Full)
		if err != nil {
			contextlog.LogErrorWithContext(redirectCtx, logging.GetSlogger(), "Failed to insert redirect URL: %v", err)
			return err
		}
		contextlog.LogDebugWithContext(redirectCtx, logging.GetSlogger(), "Saved redirection URL %s", newURL.String())
	}
	return nil
}

//func GetSnapshotFromURL(tx *sqlx.Tx, url string) ([]snapshot.Snapshot, error) {
//	query := `
//	SELECT *
//	FROM snapshots
//	WHERE url=$1
//	LIMIT 1
//	`
//	var snapshots []snapshot.Snapshot
//	err := tx.Select(&snapshots, query, url)
//	if err != nil {
//		return nil, err
//	}
//	return snapshots, nil
//}
