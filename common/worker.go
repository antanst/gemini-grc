package common

import (
	"fmt"
	"time"

	"gemini-grc/common/blackList"
	errors2 "gemini-grc/common/errors"
	"gemini-grc/common/snapshot"
	url2 "gemini-grc/common/url"
	_db "gemini-grc/db"
	"gemini-grc/errors"
	"gemini-grc/gemini"
	"gemini-grc/gopher"
	"gemini-grc/hostPool"
	"gemini-grc/logging"
	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

func CrawlOneURL(db *sqlx.DB, url *string) error {
	parsedURL, err := url2.ParseURL(*url, "", true)
	if err != nil {
		return err
	}

	if !url2.IsGeminiUrl(parsedURL.String()) && !url2.IsGopherURL(parsedURL.String()) {
		return errors.NewError(fmt.Errorf("error parsing URL: not a Gemini or Gopher URL: %s", parsedURL.String()))
	}

	tx, err := db.Beginx()
	if err != nil {
		return errors.NewFatalError(err)
	}

	err = _db.InsertURL(tx, parsedURL.Full)
	if err != nil {
		return err
	}

	err = workOnUrl(0, tx, parsedURL.Full)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		//if _db.IsDeadlockError(err) {
		//	logging.LogError("Deadlock detected. Rolling back")
		//	time.Sleep(time.Duration(10) * time.Second)
		//	err := tx.Rollback()
		//	return errors.NewFatalError(err)
		//}
		return errors.NewFatalError(err)
	}
	logging.LogInfo("Done")
	return nil
}

func SpawnWorkers(numOfWorkers int, db *sqlx.DB) {
	logging.LogInfo("Spawning %d workers", numOfWorkers)
	go PrintWorkerStatus(numOfWorkers, StatusChan)

	for i := range numOfWorkers {
		go func(i int) {
			UpdateWorkerStatus(i, "Waiting to start")
			// Jitter to avoid starting everything at the same time
			time.Sleep(time.Duration(i+2) * time.Second)
			for {
				// TODO: Use cancellable context with tx, logger & worker ID.
				// ctx := context.WithCancel()
				// ctx = context.WithValue(ctx, common.CtxKeyLogger, &RequestLogger{r: r})
				RunWorkerWithTx(i, db)
			}
		}(i)
	}
}

func RunWorkerWithTx(workerID int, db *sqlx.DB) {
	defer func() {
		UpdateWorkerStatus(workerID, "Done")
	}()

	tx, err := db.Beginx()
	if err != nil {
		ErrorsChan <- err
		return
	}

	err = runWorker(workerID, tx)
	if err != nil {
		// TODO: Rollback in this case?
		ErrorsChan <- err
		return
	}

	logging.LogDebug("[%3d] Committing transaction", workerID)
	err = tx.Commit()
	// On deadlock errors, rollback and return, otherwise panic.
	if err != nil {
		logging.LogError("[%3d] Failed to commit transaction: %w", workerID, err)
		if _db.IsDeadlockError(err) {
			logging.LogError("[%3d] Deadlock detected. Rolling back", workerID)
			time.Sleep(time.Duration(10) * time.Second)
			err := tx.Rollback()
			if err != nil {
				panic(fmt.Sprintf("[%3d] Failed to roll back transaction: %v", workerID, err))
			}
			return
		}
		panic(fmt.Sprintf("[%3d] Failed to commit transaction: %v", workerID, err))
	}
	logging.LogDebug("[%3d] Worker done!", workerID)
}

func runWorker(workerID int, tx *sqlx.Tx) error {
	var urls []string
	var err error

	UpdateWorkerStatus(workerID, "Getting URLs from DB")
	urls, err = _db.GetRandomUrls(tx)
	// urls, err = _db.GetRandomUrlsWithBasePath(tx)
	if err != nil {
		return err
	} else if len(urls) == 0 {
		logging.LogInfo("[%3d] No URLs to visit, sleeping...", workerID)
		UpdateWorkerStatus(workerID, "No URLs to visit, sleeping...")
		time.Sleep(1 * time.Minute)
		return nil
	}

	// Start visiting URLs.
	total := len(urls)
	for i, u := range urls {
		logging.LogInfo("[%3d] Starting %d/%d %s", workerID, i+1, total, u)
		UpdateWorkerStatus(workerID, fmt.Sprintf("Starting %d/%d %s", i+1, total, u))
		err := workOnUrl(workerID, tx, u)
		if err != nil {
			return err
		}
		logging.LogDebug("[%3d] Done %d/%d.", workerID, i+1, total)
		UpdateWorkerStatus(workerID, fmt.Sprintf("Done %d/%d %s", i+1, total, u))
	}
	return nil
}

// workOnUrl visits a URL and stores the result.
// unexpected errors are returned.
// expected errors are stored within the snapshot.
func workOnUrl(workerID int, tx *sqlx.Tx, url string) (err error) {
	s, err := snapshot.SnapshotFromURL(url, false)
	if err != nil {
		return err
	}

	isGemini := url2.IsGeminiUrl(s.URL.String())
	isGopher := url2.IsGopherURL(s.URL.String())
	if !isGemini && !isGopher {
		return errors.NewError(fmt.Errorf("not a Gopher or Gemini URL: %s", s.URL.String()))
	}

	if blackList.IsBlacklisted(s.URL.String()) {
		logging.LogInfo("[%3d] URL matches blacklist, ignoring", workerID)
		s.Error = null.StringFrom(errors2.ErrBlacklistMatch.Error())
		return saveSnapshotAndRemoveURL(tx, s)
	}

	if isGemini {
		// If URL matches a robots.txt disallow line,
		// add it as an error and remove url
		robotMatch, err := gemini.RobotMatch(s.URL.String())
		if err != nil {
			// robotMatch returns only network errors!
			// we stop because we don't want to hit
			// the server with another request on this case.
			return err
		}
		if robotMatch {
			logging.LogInfo("[%3d] URL matches robots.txt, ignoring", workerID)
			s.Error = null.StringFrom(errors2.ErrRobotsMatch.Error())
			return saveSnapshotAndRemoveURL(tx, s)
		}
	}

	logging.LogDebug("[%3d] Adding to pool %s", workerID, s.URL.String())
	UpdateWorkerStatus(workerID, fmt.Sprintf("Adding to pool %s", s.URL.String()))
	hostPool.AddHostToHostPool(s.Host)
	defer func(s string) {
		hostPool.RemoveHostFromPool(s)
	}(s.Host)

	logging.LogDebug("[%3d] Visiting %s", workerID, s.URL.String())
	UpdateWorkerStatus(workerID, fmt.Sprintf("Visiting %s", s.URL.String()))

	if isGopher {
		s, err = gopher.Visit(s.URL.String())
	} else {
		s, err = gemini.Visit(s.URL.String())
	}

	if err != nil {
		return err
	}

	// Handle Gemini redirection.
	if isGemini &&
		s.ResponseCode.ValueOrZero() >= 30 &&
		s.ResponseCode.ValueOrZero() < 40 {
		err = handleRedirection(workerID, tx, s)
		if err != nil {
			return fmt.Errorf("error while handling redirection: %s", err)
		}
	}

	// Store links
	if len(s.Links.ValueOrZero()) > 0 {
		logging.LogDebug("[%3d] Found %d links", workerID, len(s.Links.ValueOrZero()))
		err = storeLinks(tx, s)
		if err != nil {
			return err
		}
	}

	logging.LogInfo("[%3d] %2d %s", workerID, s.ResponseCode.ValueOrZero(), s.URL.String())
	return saveSnapshotAndRemoveURL(tx, s)
}

func storeLinks(tx *sqlx.Tx, s *snapshot.Snapshot) error {
	if s.Links.Valid { //nolint:nestif
		for _, link := range s.Links.ValueOrZero() {
			if shouldPersistURL(&link) {
				visited, err := haveWeVisitedURL(tx, link.Full)
				if err != nil {
					return err
				}
				if !visited {
					err := _db.InsertURL(tx, link.Full)
					if err != nil {
						return err
					}
				} else {
					logging.LogDebug("Link already persisted: %s", link.Full)
				}
			}
		}
	}
	return nil
}

func saveSnapshotAndRemoveURL(tx *sqlx.Tx, s *snapshot.Snapshot) error {
	err := _db.OverwriteSnapshot(tx, s)
	if err != nil {
		return err
	}
	err = _db.DeleteURL(tx, s.URL.String())
	if err != nil {
		return err
	}
	return nil
}

// shouldPersistURL returns true if we
// should save the URL in the _db.
// Only gemini:// urls are saved.
func shouldPersistURL(u *url2.URL) bool {
	return url2.IsGeminiUrl(u.String()) || url2.IsGopherURL(u.String())
}

func haveWeVisitedURL(tx *sqlx.Tx, u string) (bool, error) {
	var result []bool
	err := tx.Select(&result, `SELECT TRUE FROM urls WHERE url=$1`, u)
	if err != nil {
		return false, errors.NewFatalError(fmt.Errorf("database error: %w", err))
	}
	if len(result) > 0 {
		return result[0], nil
	}
	err = tx.Select(&result, `SELECT TRUE FROM snapshots WHERE snapshots.url=$1`, u)
	if err != nil {
		return false, errors.NewFatalError(fmt.Errorf("database error: %w", err))
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return false, nil
}

// handleRedirection saves redirection URL.
func handleRedirection(workerID int, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	newURL, err := url2.ExtractRedirectTargetFromHeader(s.URL, s.Error.ValueOrZero())
	if err != nil {
		return err
	}
	logging.LogDebug("[%3d] Page redirects to %s", workerID, newURL)

	haveWeVisited, _ := haveWeVisitedURL(tx, newURL.String())
	if shouldPersistURL(newURL) && !haveWeVisited {
		err = _db.InsertURL(tx, newURL.Full)
		if err != nil {
			return err
		}
		logging.LogDebug("[%3d] Saved redirection URL %s", workerID, newURL.String())
	}
	return nil
}

func GetSnapshotFromURL(tx *sqlx.Tx, url string) ([]snapshot.Snapshot, error) {
	query := `
	SELECT *
	FROM snapshots
	WHERE url=$1
	LIMIT 1
	`
	var snapshots []snapshot.Snapshot
	err := tx.Select(&snapshots, query, url)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
