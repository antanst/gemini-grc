package gemini

import (
	"fmt"
	"gemini-grc/config"
	"gemini-grc/logging"
	"gemini-grc/uid"
	"gemini-grc/util"
	"strings"
	"time"

	"github.com/guregu/null/v5"
	"github.com/jmoiron/sqlx"
)

func SpawnWorkers(numOfWorkers int, db *sqlx.DB) {
	logging.LogInfo("Spawning %d workers", numOfWorkers)
	for i := 0; i < numOfWorkers; i++ {
		go func(i int) {
			for {
				runWorker(i, db)
			}
		}(i)
	}
}

func runWorker(id int, db *sqlx.DB) {
	// Start the DB transaction
	tx, err := db.Beginx()
	if err != nil {
		logging.LogError("Failed to begin transaction: %w", err)
	}

	defer func() {
		err = tx.Commit()
		if err != nil {
			logging.LogError("[%d] Failed to commit transaction: %w", id, err)
			err := tx.Rollback()
			if err != nil {
				panic(fmt.Sprintf("[%d] Failed to roll back transaction: %v", id, err))
			}
		}
	}()

	snapshots, err := GetRandomSnapshotsDistinctHosts(tx)

	if err != nil {
		logging.LogError("[%d] Error retrieving snapshot: %w", id, err)
		time.Sleep(10 * time.Second)
		return
	} else if len(snapshots) == 0 {
		logging.LogInfo("[%d] No remaining snapshots to visit.", id)
		time.Sleep(1 * time.Minute)
		return
	}
	total := len(snapshots)
	for i, s := range snapshots {
		logging.LogInfo("[%d] Starting %d/%d %s", id, i+1, total, s.URL)
		err = workOnSnapshot(id, tx, &s)
		if err != nil {
			logging.LogError("[%d] [%s] Unexpected Error %w", id, s.URL, err)
			util.PrintStackAndPanic(err)
		}
		if s.Error.Valid {
			logging.LogWarn("[%d] [%s] Worker Error: %v", id, s.URL, s.Error.String)
		}
		logging.LogDebug("[%d] Done %d/%d.", id, i, total)
	}
	logging.LogInfo("[%d] Worker done.", id)
}

func workOnSnapshot(id int, tx *sqlx.Tx, s *Snapshot) (err error) {
	// If URL matches a robots.txt disallow line,
	// add it as an error so next time it won't be
	// crawled.
	if RobotMatch(s) {
		s.Error = null.StringFrom("robots.txt disallow match")
		err = SaveSnapshotToDB(tx, s)
		if err != nil {
			return fmt.Errorf("[%d] DB Error: %w", id, err)
		}
		return nil
	}

	IPs, err := getHostIPAddresses(s.Host)
	if err != nil {
		s.Error = null.StringFrom("DNS Resolve error")
		err = SaveSnapshotToDB(tx, s)
		if err != nil {
			return fmt.Errorf("[%d] DB Error: %w", id, err)
		}
		return nil
	}

	// If the host's ip is in the connections pool,
	// stop and add the url in the queue later.
	IpPool.Lock.RLock()
	logging.LogDebug("[%d] [%s] Checking pool for IP", id, s.URL)
	for _, ip := range IPs {
		_, ok := IpPool.IPs[ip]
		if ok {
			logging.LogDebug("[%d] Another worker is visiting this host: %s", id, s.URL)
			IpPool.Lock.RUnlock()
			time.Sleep(1 * time.Second) // Avoid flood-retrying when few URLs remain
			return nil
		}
	}
	IpPool.Lock.RUnlock()

	AddIPsToPool(IPs)

	url := s.URL.String()
	logging.LogDebug("[%d] Dialing %s", id, url)
	Visit(s)
	logging.LogDebug("[%d] Finished dialing.", id)

	go func() {
		time.Sleep(5 * time.Second)
		RemoveIPsFromPool(IPs)
	}()

	if s.MimeType.Valid && s.MimeType.String == "text/gemini" {
		logging.LogDebug("[%d] [%s] Processing", id, url)
		s = ProcessGemini(s)
	}
	logging.LogDebug("[%d] Saving", id)
	err = SaveSnapshotToDB(tx, s)
	if err != nil {
		return fmt.Errorf("[%d] DB Error: %w", id, err)
	}

	// Store links in batch
	if s.Links != nil {
		var batchSnapshots []*Snapshot
		timestamp := null.TimeFrom(time.Now())

		for _, link := range *s.Links {
			if shouldPersistURL(tx, link) {
				newSnapshot := &Snapshot{
					UID:       uid.UID(),
					URL:       link,
					Host:      link.Hostname,
					Timestamp: timestamp,
				}
				batchSnapshots = append(batchSnapshots, newSnapshot)
			}
		}

		if len(batchSnapshots) > 0 {
			logging.LogDebug("[%d] Batch saving %d links", id, len(batchSnapshots))
			err = SaveLinksToDBinBatches(tx, batchSnapshots)
			if err != nil {
				return fmt.Errorf("[%d] DB Error: %w", id, err)
			}
		}
	}
	return nil
}

// Should we save the given URL for crawling?
func shouldPersistURL(tx *sqlx.Tx, u GeminiUrl) bool {
	if !strings.HasPrefix(u.String(), "gemini://") {
		return false
	}
	query := `SELECT EXISTS(SELECT 1 FROM snapshots WHERE URL=$1)`
	var exists bool
	err := tx.Get(&exists, query, u.String())
	if err != nil {
		fmt.Println("Error executing query:", err)
		return false
	}
	return !exists
}

func GetRandomSnapshotsDistinctHosts(tx *sqlx.Tx) ([]Snapshot, error) {
	// Old, unoptimized query
	//
	// 	query := `
	//     SELECT DISTINCT ON (host) *
	//     FROM snapshots
	//     WHERE response_code IS NULL
	//       AND error IS NULL
	//     ORDER BY host, RANDOM()
	//     LIMIT $1
	// `
	query := `
        WITH RankedSnapshots AS (
            SELECT id, uid, url, host, timestamp, mimetype, data, gemtext,
                   links, lang, response_code, error,
                   ROW_NUMBER() OVER (PARTITION BY host ORDER BY RANDOM()) as rn
            FROM snapshots
            WHERE response_code IS NULL
              AND error IS NULL
        )
        SELECT id, uid, url, host, timestamp, mimetype, data, gemtext,
               links, lang, response_code, error
        FROM RankedSnapshots
        WHERE rn = 1
        LIMIT $1
    `
	var snapshots []Snapshot
	err := tx.Select(&snapshots, query, config.CONFIG.WorkerBatchSize)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
