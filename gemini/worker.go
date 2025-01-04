package gemini

import (
	"errors"
	"fmt"
	"gemini-grc/common"
	_db "gemini-grc/db"
	"strings"
	"time"

	"gemini-grc/logging"
	"gemini-grc/util"
	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

func SpawnWorkers(numOfWorkers int, db *sqlx.DB) {
	logging.LogInfo("Spawning %d workers", numOfWorkers)
	statusChan = make(chan WorkerStatus, numOfWorkers)
	go PrintWorkerStatus(numOfWorkers, statusChan)

	for i := range numOfWorkers {
		go func(i int) {
			// Jitter to avoid starting everything at the same time
			time.Sleep(time.Duration(util.SecureRandomInt(10)) * time.Second)
			for {
				RunWorkerWithTx(i, db, nil)
			}
		}(i)
	}
}

func RunWorkerWithTx(workerID int, db *sqlx.DB, url *string) {
	statusChan <- WorkerStatus{
		id:     workerID,
		status: "Starting up",
	}
	defer func() {
		statusChan <- WorkerStatus{
			id:     workerID,
			status: "Done",
		}
	}()
	tx, err := db.Beginx()
	if err != nil {
		panic(fmt.Sprintf("Failed to begin transaction: %v", err))
	}
	runWorker(workerID, tx, url)
	logging.LogDebug("[%d] Committing transaction", workerID)
	err = tx.Commit()
	// On deadlock errors, rollback and return, otherwise panic.
	if err != nil {
		logging.LogError("[%d] Failed to commit transaction: %w", workerID, err)
		if _db.IsDeadlockError(err) {
			logging.LogError("[%d] Deadlock detected. Rolling back", workerID)
			time.Sleep(time.Duration(10) * time.Second)
			err := tx.Rollback()
			if err != nil {
				panic(fmt.Sprintf("[%d] Failed to roll back transaction: %v", workerID, err))
			}
			return
		}
		panic(fmt.Sprintf("[%d] Failed to commit transaction: %v", workerID, err))
	}
	logging.LogDebug("[%d] Worker done!", workerID)
}

func runWorker(workerID int, tx *sqlx.Tx, url *string) {
	var urls []string
	var err error

	// If not given a specific URL,
	// get some random ones to visit from db.
	if url == nil {
		statusChan <- WorkerStatus{
			id:     workerID,
			status: "Getting URLs",
		}
		urls, err = _db.GetURLsToVisit(tx)
		if err != nil {
			logging.LogError("[%d] GeminiError retrieving snapshot: %w", workerID, err)
			panic("This should never happen")
		} else if len(urls) == 0 {
			logging.LogInfo("[%d] No URLs to visit.", workerID)
			time.Sleep(1 * time.Minute)
			return
		}
	} else {
		geminiURL, err := common.ParseURL(*url, "")
		if err != nil {
			logging.LogError("Invalid URL given: %s", *url)
			return
		}
		urls = []string{geminiURL.String()}
	}

	// Start visiting URLs.
	total := len(urls)
	for i, u := range urls {
		logging.LogDebug("[%d] Starting %d/%d %s", workerID, i+1, total, u)
		// We differentiate between errors:
		// Unexpected errors are the ones returned from the following function.
		// If an error is unexpected (which should never happen) we panic.
		// Expected errors are stored as strings within the snapshot.
		err := workOnUrl(workerID, tx, u)
		if err != nil {
			logging.LogError("[%d] Unexpected GeminiError %w while visiting %s", workerID, err, u)
			util.PrintStackAndPanic(err)
		}
		logging.LogDebug("[%d] Done %d/%d.", workerID, i+1, total)
	}
}

// workOnUrl visits a URL and stores the result.
// unexpected errors are returned.
// expected errors are stored within the snapshot.
func workOnUrl(workerID int, tx *sqlx.Tx, url string) (err error) {
	if url == "" {
		return fmt.Errorf("nil URL given")
	}

	if IsBlacklisted(url) {
		logging.LogDebug("[%d] URL matches Blacklist, ignoring %s", workerID, url)
		return nil
	}

	s := common.SnapshotFromURL(url)

	// If URL matches a robots.txt disallow line,
	// add it as an error so next time it won't be
	// crawled.
	if RobotMatch(url) {
		s.Error = null.StringFrom(common.ErrGeminiRobotsDisallowed.Error())
		err = _db.OverwriteSnapshot(workerID, tx, s)
		if err != nil {
			return fmt.Errorf("[%d] %w", workerID, err)
		}
		return nil
	}

	// Resolve IP address via DNS
	statusChan <- WorkerStatus{
		id:     workerID,
		status: fmt.Sprintf("Resolving %s", url),
	}
	IPs, err := getHostIPAddresses(s.Host)
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		err = _db.OverwriteSnapshot(workerID, tx, s)
		if err != nil {
			return fmt.Errorf("[%d] %w", workerID, err)
		}
		return nil
	}

	for {
		count := 1
		if isAnotherWorkerVisitingHost(workerID, IPs) {
			logging.LogDebug("[%d] Another worker is visiting this host, waiting", workerID)
			statusChan <- WorkerStatus{
				id:     workerID,
				status: fmt.Sprintf("Waiting to grab lock for host %s", s.Host),
			}
			time.Sleep(2 * time.Second) // Avoid flood-retrying
			count++
			if count == 3 {
				return
			}
		} else {
			break
		}
	}

	statusChan <- WorkerStatus{
		id:     workerID,
		status: fmt.Sprintf("Adding to pool %s", url),
	}
	AddIPsToPool(IPs)
	// After finishing, remove the host IPs from
	// the connections pool, with a small delay
	// to avoid potentially hitting the same IP quickly.
	defer func() {
		go func() {
			time.Sleep(1 * time.Second)
			statusChan <- WorkerStatus{
				id:     workerID,
				status: fmt.Sprintf("Removing from pool %s", url),
			}
			RemoveIPsFromPool(IPs)
		}()
	}()

	statusChan <- WorkerStatus{
		id:     workerID,
		status: fmt.Sprintf("Visiting %s", url),
	}

	err = Visit(s)
	if err != nil {
		if !common.IsKnownError(err) {
			logging.LogError("[%d] Unknown error visiting %s: %w", workerID, url, err)
			return err
		}
		s.Error = null.StringFrom(err.Error())
		// Check if error is redirection, and handle it
		if errors.As(err, new(*common.GeminiError)) &&
		    err.(*common.GeminiError).Msg == "redirect" {
			err = handleRedirection(workerID, tx, s)
			if err != nil {
				if common.IsKnownError(err) {
					s.Error = null.StringFrom(err.Error())
				} else {
					return err
				}
			}
		}
	}
	// If this is a gemini page, parse possible links inside
	if !s.Error.Valid && s.MimeType.Valid && s.MimeType.String == "text/gemini" {
		links := GetPageLinks(s.URL, s.GemText.String)
		if len(links) > 0 {
			logging.LogDebug("[%d] Found %d links", workerID, len(links))
			s.Links = null.ValueFrom(links)
			err = storeLinks(tx, s)
			if err != nil {
				return err
			}
		}
	} else {
		logging.LogDebug("[%d] Not text/gemini, so not looking for page links", workerID)
	}

	err = _db.OverwriteSnapshot(workerID, tx, s)
	logging.LogInfo("[%3d] %2d %s", workerID, s.ResponseCode.ValueOrZero(), s.URL.String())
	if err != nil {
		return err
	}

	return nil
}

func isAnotherWorkerVisitingHost(workerID int, IPs []string) bool {
	IPPool.Lock.RLock()
	defer func() {
		IPPool.Lock.RUnlock()
	}()
	logging.LogDebug("[%d] Checking pool for IPs", workerID)
	for _, ip := range IPs {
		_, ok := IPPool.IPs[ip]
		if ok {
			return true
		}
	}
	return false
}

func storeLinks(tx *sqlx.Tx, s *common.Snapshot) error {
	if s.Links.Valid {
		var batchSnapshots []*common.Snapshot
		for _, link := range s.Links.ValueOrZero() {
			if shouldPersistURL(&link) {
				newSnapshot := &common.Snapshot{
					URL:       link,
					Host:      link.Hostname,
					Timestamp: null.TimeFrom(time.Now()),
				}
				batchSnapshots = append(batchSnapshots, newSnapshot)
			}
		}

		if len(batchSnapshots) > 0 {
			err := _db.SaveLinksToDBinBatches(tx, batchSnapshots)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldPersistURL returns true if we
// should save the URL in the _db.
// Only gemini:// urls are saved.
func shouldPersistURL(u *common.URL) bool {
	return strings.HasPrefix(u.String(), "gemini://")
}

func haveWeVisitedURL(tx *sqlx.Tx, u *common.URL) (bool, error) {
	var result bool
	err := tx.Select(&result, `SELECT TRUE FROM urls WHERE url=$1`, u.String())
	if err != nil {
		return false, fmt.Errorf("%w: %w", common.ErrDatabase, err)
	}
	if result {
		return result, nil
	}
	err = tx.Select(&result, `SELECT TRUE FROM snapshots WHERE snapshot.url=$1`, u.String())
	if err != nil {
		return false, fmt.Errorf("%w: %w", common.ErrDatabase, err)
	}
	return result, nil
}

// handleRedirection saves redirect URL as new snapshot
func handleRedirection(workerID int, tx *sqlx.Tx, s *common.Snapshot) error {
	newURL, err := extractRedirectTarget(s.URL, s.Error.ValueOrZero())
	if err != nil {
		if errors.Is(err, common.ErrGeminiRedirect) {
			logging.LogDebug("[%d] %s", workerID, err)
		}
		return err
	}
	logging.LogDebug("[%d] Page redirects to %s", workerID, newURL)
	// Insert fresh snapshot with new URL
	if shouldPersistURL(newURL) {
		snapshot := &common.Snapshot{
			// UID:       uid.UID(),
			URL:       *newURL,
			Host:      newURL.Hostname,
			Timestamp: null.TimeFrom(time.Now()),
		}
		logging.LogDebug("[%d] Saving redirection URL %s", workerID, snapshot.URL.String())
		err = _db.SaveSnapshotIfNew(tx, snapshot)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetSnapshotFromURL(tx *sqlx.Tx, url string) ([]common.Snapshot, error) {
	query := `
	SELECT *
	FROM snapshots
	WHERE url=$1
	LIMIT 1
	`
	var snapshots []common.Snapshot
	err := tx.Select(&snapshots, query, url)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
