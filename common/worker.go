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
	"gemini-grc/robotsMatch"
	"git.antanst.com/antanst/logging"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

func RunWorkerWithTx(workerID int, job string) {
	parsedURL, err := url2.ParseURL(job, "", true)
	if err != nil {
		logging.LogInfo("Failed to parse URL: %s Error: %s", job, err)
		return
	}
	host := parsedURL.Hostname

	// Create a new worker context
	baseCtx := context.Background()
	ctx, cancel := contextutil.NewRequestContext(baseCtx, job, host, workerID)
	ctx = contextutil.ContextWithComponent(ctx, "worker")
	defer cancel() // Ensure the context is cancelled when we're done
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Starting worker for URL %s", job)

	// Create a new db transaction
	tx, err := gemdb.Database.NewTx(ctx)
	if err != nil {
		FatalErrorsChan <- err
		return
	}

	err = runWorker(ctx, tx, []string{job})
	WorkerWG.Done()
	if err != nil {
		// Two cases to handle:
		// - context cancellation/timeout errors (log and ignore)
		// - fatal errors (log and send to chan)
		// non-fatal errors should've been handled within
		// the runWorker() function and not bubble up here.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Worker timed out or canceled: %v", err)
			rollbackErr := gemdb.SafeRollback(ctx, tx)
			if rollbackErr != nil {
				FatalErrorsChan <- rollbackErr
				return
			}
			return
		} else if xerrors.IsFatal(err) {
			contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Worker failed: %v", err)
			rollbackErr := gemdb.SafeRollback(ctx, tx)
			if rollbackErr != nil {
				FatalErrorsChan <- rollbackErr
				return
			}
			FatalErrorsChan <- err
			return

		}
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Worker failed: %v", err)
		rollbackErr := gemdb.SafeRollback(ctx, tx)
		if rollbackErr != nil {
			FatalErrorsChan <- rollbackErr
			return
		}
		return
	}

	err = tx.Commit()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to commit transaction: %v", err)
		if rollbackErr := gemdb.SafeRollback(ctx, tx); rollbackErr != nil {
			FatalErrorsChan <- err
			return
		}
	}
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Worker done.")
}

func runWorker(ctx context.Context, tx *sqlx.Tx, urls []string) error {
	for _, u := range urls {
		err := WorkOnUrl(ctx, tx, u)
		if err != nil {
			return err
		}
	}
	return nil
}

// WorkOnUrl visits a URL and stores the result.
// unexpected errors are returned.
// expected errors are stored within the snapshot.
func WorkOnUrl(ctx context.Context, tx *sqlx.Tx, url string) (err error) {
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Worker visiting URL %s", url)

	s, err := snapshot.SnapshotFromURL(url, true)
	if err != nil {
		return err
	}

	// We always use the normalized URL
	if url != s.URL.Full {
		url = s.URL.Full
	}

	isGemini := url2.IsGeminiUrl(s.URL.String())
	isGopher := url2.IsGopherURL(s.URL.String())

	if !isGemini && !isGopher {
		return xerrors.NewSimpleError(fmt.Errorf("not a Gopher or Gemini URL: %s", s.URL.String()))
	}

	if isGopher && !config.CONFIG.GopherEnable {
		return xerrors.NewSimpleError(fmt.Errorf("gopher disabled, not processing Gopher URL: %s", s.URL.String()))
	}

	// Check if URL is whitelisted
	isUrlWhitelisted := whiteList.IsWhitelisted(s.URL.String())
	if isUrlWhitelisted {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "URL matches whitelist, forcing crawl %s", url)
	}

	// Only check blacklist if URL is not whitelisted
	if !isUrlWhitelisted && blackList.IsBlacklisted(s.URL.String()) {
		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "URL matches blacklist, skipped")
		s.Error = null.StringFrom(commonErrors.ErrBlacklistMatch.Error())
		return saveSnapshotAndRemoveURL(ctx, tx, s)
	}

	// Only check robots.txt if URL is not whitelisted and is a Gemini URL
	var robotMatch bool
	if !isUrlWhitelisted && isGemini {
		// If URL matches a robots.txt disallow line,
		// add it as an error and remove url
		robotMatch = robotsMatch.RobotMatch(ctx, s.URL.String())
		if robotMatch {
			contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "URL matches robots.txt, skipped")
			s.Error = null.StringFrom(commonErrors.ErrRobotsMatch.Error())
			return saveSnapshotAndRemoveURL(ctx, tx, s)
		}
	}

	err = hostPool.AddHostToHostPool(ctx, s.Host)
	if err != nil {
		return err
	}

	defer func(ctx context.Context, host string) {
		hostPool.RemoveHostFromPool(ctx, host)
	}(ctx, s.Host)

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Visiting %s", s.URL.String())

	// Use context-aware visits for both protocols
	if isGopher {
		s, err = gopher.VisitWithContext(ctx, s.URL.String())
	} else {
		s, err = gemini.Visit(ctx, s.URL.String())
	}

	if err != nil {
		return err
	}

	// Handle Gemini redirection.
	if isGemini &&
		s.ResponseCode.ValueOrZero() >= 30 &&
		s.ResponseCode.ValueOrZero() < 40 {
		err = saveRedirectURL(ctx, tx, s)
		if err != nil {
			return xerrors.NewSimpleError(fmt.Errorf("error while handling redirection: %s", err))
		}
	}

	// Check if we should skip a potentially
	// identical snapshot with one from history
	isIdentical, err := isContentIdentical(ctx, tx, s)
	if err != nil {
		return err
	}
	if isIdentical {
		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "Content identical to existing snapshot, updating crawl timestamp")
		// Update the last_crawled timestamp to track that we processed this URL
		err = gemdb.Database.UpdateLastCrawled(ctx, tx, s.URL.String())
		if err != nil {
			return err
		}
		return removeURL(ctx, tx, s.URL.String())
	}

	// Process and store links since content has changed
	if len(s.Links.ValueOrZero()) > 0 {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Found %d links", len(s.Links.ValueOrZero()))
		err = storeLinks(ctx, tx, s)
		if err != nil {
			return err
		}
	}

	return saveSnapshotAndRemoveURL(ctx, tx, s)
}

func shouldUpdateSnapshotData(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) (bool, error) {
	// If we don't have an error, save the new snapshot.
	if !s.Error.Valid {
		return true, nil
	}
	prevSnapshot, err := gemdb.Database.GetLatestSnapshot(ctx, tx, s.URL.String())
	if err != nil {
		return false, err
	}
	// If we don't have a previous snapshot, save it anyway.
	if prevSnapshot == nil {
		return true, nil
	}
	// If we have a previous snapshot,
	// and it didn't have an error, save.
	// This means that we can have a max
	// of one consecutive snapshot with
	// an error.
	if prevSnapshot.Error.ValueOrZero() == "" {
		return true, nil
	}
	return false, nil
}

func isContentIdentical(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) (bool, error) {
	// Always check if content is identical to previous snapshot
	identical, err := gemdb.Database.IsContentIdentical(ctx, tx, s)
	if err != nil {
		return false, err
	}
	return identical, nil
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

func removeURL(ctx context.Context, tx *sqlx.Tx, url string) error {
	return gemdb.Database.DeleteURL(ctx, tx, url)
}

func saveSnapshotAndRemoveURL(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	shouldUpdateSnapshot, err := shouldUpdateSnapshotData(ctx, tx, s)
	if err != nil {
		return err
	}
	if shouldUpdateSnapshot {
		err := gemdb.Database.SaveSnapshot(ctx, tx, s)
		if err != nil {
			return err
		}
		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "%2d", s.ResponseCode.ValueOrZero())
		return removeURL(ctx, tx, s.URL.String())
	} else {
		contextlog.LogInfoWithContext(ctx, logging.GetSlogger(), "%2d (but old content exists, updating crawl date)", s.ResponseCode.ValueOrZero())
		err = gemdb.Database.UpdateLastCrawled(ctx, tx, s.URL.String())
		if err != nil {
			return err
		}
		return removeURL(ctx, tx, s.URL.String())
	}
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
		return false, xerrors.NewSimpleError(err)
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
			return true, nil
		}
	}

	return false, nil
}

func saveRedirectURL(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	newURL, err := url2.ExtractRedirectTargetFromHeader(s.URL, s.Header.ValueOrZero())
	if err != nil {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Failed to extract redirect target: %v", err)
		return err
	}
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Page redirects to %s", newURL)

	haveWeVisited, err := haveWeVisitedURL(ctx, tx, newURL.String())
	if err != nil {
		return err
	}

	if shouldPersistURL(newURL) && !haveWeVisited {
		err = gemdb.Database.InsertURL(ctx, tx, newURL.Full)
		if err != nil {
			return err
		}
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Saved redirection URL %s", newURL.String())
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
