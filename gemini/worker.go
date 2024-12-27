package gemini

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gemini-grc/logging"
	"gemini-grc/util"
	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

type WorkerStatus struct {
	id     int
	status string
}

func PrintWorkerStatus(totalWorkers int, statusChan chan WorkerStatus) {
	// Create a slice to store current status of each worker
	statuses := make([]string, totalWorkers)

	// Initialize empty statuses
	for i := range statuses {
		statuses[i] = ""
	}

	// Initial print
	var output strings.Builder
	// \033[H moves the cursor to the top left corner of the screen
	// (ie, the first column of the first row in the screen).
	// \033[J clears the part of the screen from the cursor to the end of the screen.
	output.WriteString("\033[H\033[J") // Clear screen and move cursor to top
	for i := range statuses {
		output.WriteString(fmt.Sprintf("[%2d] \n", i))
	}
	fmt.Print(output.String())

	// Continuously receive status updates
	for update := range statusChan {
		if update.id >= totalWorkers {
			continue
		}

		// Update the status
		statuses[update.id] = update.status

		// Build the complete output string
		output.Reset()
		output.WriteString("\033[H\033[J") // Clear screen and move cursor to top
		for i, status := range statuses {
			output.WriteString(fmt.Sprintf("[%2d] %.100s\n", i, status))
		}

		// Print the entire status
		fmt.Print(output.String())
	}
}

var statusChan chan WorkerStatus

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
		if isDeadlockError(err) {
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
	var snapshots []Snapshot
	var err error

	// If not given a specific URL,
	// get some random ones to visit from DB.
	if url == nil {
		statusChan <- WorkerStatus{
			id:     workerID,
			status: "Getting snapshots",
		}
		snapshots, err = GetSnapshotsToVisit(tx)
		if err != nil {
			logging.LogError("[%d] GeminiError retrieving snapshot: %w", workerID, err)
			panic("This should never happen")
		} else if len(snapshots) == 0 {
			logging.LogInfo("[%d] No snapshots to visit.", workerID)
			time.Sleep(1 * time.Minute)
			return
		}
	} else {
		snapshotURL, err := ParseURL(*url, "")
		if err != nil {
			logging.LogError("Invalid URL given: %s", *url)
			return
		}
		snapshots = []Snapshot{{
			// UID:       uid.UID(),
			URL:       *snapshotURL,
			Host:      snapshotURL.Hostname,
			Timestamp: null.TimeFrom(time.Now()),
		}}
	}

	total := len(snapshots)
	for i, s := range snapshots {
		logging.LogDebug("[%d] Snapshot %d/%d: %s", workerID, i+1, total, s.URL.String())
	}
	// Start visiting URLs.
	for i, s := range snapshots {
		logging.LogDebug("[%d] Starting %d/%d %s", workerID, i+1, total, s.URL.String())
		// We differentiate between errors:
		// Unexpected errors are the ones returned from the following function.
		// If an error is unexpected (which should never happen) we panic.
		// Expected errors are stored as strings within the snapshot,
		// so that they can also be stored in DB.
		err := workOnSnapshot(workerID, tx, &s)
		if err != nil {
			logging.LogError("[%d] [%s] Unexpected GeminiError %w", workerID, s.URL.String(), err)
			util.PrintStackAndPanic(err)
		}
		if s.Error.Valid {
			logging.LogDebug("[%d] Error: %v", workerID, s.Error.String)
		}
		logging.LogDebug("[%d] Done %d/%d.", workerID, i+1, total)
	}
}

// workOnSnapshot visits a URL and stores the result.
// unexpected errors are returned.
// expected errors are stored within the snapshot.
func workOnSnapshot(workerID int, tx *sqlx.Tx, s *Snapshot) (err error) {
	if IsBlacklisted(s.URL) {
		logging.LogDebug("[%d] URL matches Blacklist, ignoring %s", workerID, s.URL.String())
		return nil
	}

	// If URL matches a robots.txt disallow line,
	// add it as an error so next time it won't be
	// crawled.
	if RobotMatch(s.URL) {
		s.Error = null.StringFrom(ErrGeminiRobotsDisallowed.Error())
		err = UpsertSnapshot(workerID, tx, s)
		if err != nil {
			return fmt.Errorf("[%d] %w", workerID, err)
		}
		return nil
	}

	// Resolve IP address via DNS
	IPs, err := getHostIPAddresses(s.Host)
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		err = UpsertSnapshot(workerID, tx, s)
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
			time.Sleep(1 * time.Second) // Avoid flood-retrying
			count++
			if count == 3 {
				return
			}
		} else {
			break
		}
	}

	AddIPsToPool(IPs)
	// After finishing, remove the host IPs from
	// the connections pool, with a small delay
	// to avoid potentially hitting the same IP quickly.
	defer func() {
		go func() {
			time.Sleep(1 * time.Second)
			RemoveIPsFromPool(IPs)
		}()
	}()

	statusChan <- WorkerStatus{
		id:     workerID,
		status: fmt.Sprintf("Visiting %s", s.URL.String()),
	}

	err = Visit(s)
	if err != nil {
		if !IsKnownError(err) {
			logging.LogError("[%d] Unknown error visiting %s: %w", workerID, s.URL.String(), err)
			return err
		}
		s.Error = null.StringFrom(err.Error())
		// Check if error is redirection, and handle it
		if errors.As(err, new(*GeminiError)) &&
			err.(*GeminiError).Msg == "redirect" {
			err = handleRedirection(workerID, tx, s)
			if err != nil {
				if IsKnownError(err) {
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

	err = UpsertSnapshot(workerID, tx, s)
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

func storeLinks(tx *sqlx.Tx, s *Snapshot) error {
	if s.Links.Valid {
		var batchSnapshots []*Snapshot
		for _, link := range s.Links.ValueOrZero() {
			if shouldPersistURL(&link) {
				newSnapshot := &Snapshot{
					URL:       link,
					Host:      link.Hostname,
					Timestamp: null.TimeFrom(time.Now()),
				}
				batchSnapshots = append(batchSnapshots, newSnapshot)
			}
		}

		if len(batchSnapshots) > 0 {
			err := SaveLinksToDBinBatches(tx, batchSnapshots)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldPersistURL returns true if we
// should save the URL in the DB.
// Only gemini:// urls are saved.
func shouldPersistURL(u *URL) bool {
	return strings.HasPrefix(u.String(), "gemini://")
}

// handleRedirection saves redirect URL as new snapshot
func handleRedirection(workerID int, tx *sqlx.Tx, s *Snapshot) error {
	newURL, err := extractRedirectTarget(s.URL, s.Error.ValueOrZero())
	if err != nil {
		if errors.Is(err, ErrGeminiRedirect) {
			logging.LogDebug("[%d] %s", workerID, err)
		}
		return err
	}
	logging.LogDebug("[%d] Page redirects to %s", workerID, newURL)
	// Insert fresh snapshot with new URL
	if shouldPersistURL(newURL) {
		snapshot := &Snapshot{
			// UID:       uid.UID(),
			URL:       *newURL,
			Host:      newURL.Hostname,
			Timestamp: null.TimeFrom(time.Now()),
		}
		logging.LogDebug("[%d] Saving redirection URL %s", workerID, snapshot.URL.String())
		err = SaveSnapshotIfNew(tx, snapshot)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetSnapshotFromURL(tx *sqlx.Tx, url string) ([]Snapshot, error) {
	query := `
	SELECT *
	FROM snapshots
	WHERE url=$1
	LIMIT 1
	`
	var snapshots []Snapshot
	err := tx.Select(&snapshots, query, url)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
