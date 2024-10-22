package gemini

import (
	"database/sql"
	"fmt"
	"gemini-grc/config"
	"gemini-grc/logging"
	"gemini-grc/uid"
	"runtime/debug"
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

func printPoolIPs() {
	fmt.Printf("%v", IpPool.IPs)
}

func workOnSnapshot(id int, tx *sqlx.Tx, s *Snapshot) (err error) {
	// Wrap errors with more info.
	defer func() {
		if err != nil {
			err = fmt.Errorf("[%d] Worker Error: %w", id, err)
		}
	}()

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

	// Store links
	if s.Links != nil {
		for _, link := range *s.Links {
			newSnapshot := Snapshot{UID: uid.UID(), URL: link, Host: link.Hostname, Timestamp: null.TimeFrom(time.Now())}
			if shouldPersistURL(tx, link) {
				logging.LogDebug("[%d] Saving link %s", id, link)
				err = SaveLinkToDB(tx, &newSnapshot)
				if err != nil {
					return fmt.Errorf("[%d] DB Error: %w", id, err)
				}
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

// Select a random snapshot.
func GetRandomSnapshot(tx *sqlx.Tx) (*Snapshot, error) {
	query := `SELECT * FROM snapshots
              WHERE response_code IS NULL
              AND error IS NULL
	      ORDER BY RANDOM()
              LIMIT 1
              FOR UPDATE SKIP LOCKED`
	// AND (timestamp < NOW() - INTERVAL '1 day' OR timestamp IS NULL)
	var snapshot Snapshot
	err := tx.Get(&snapshot, query)
	if err != nil {
		if err == sql.ErrNoRows {
			// Handle the case where no rows were found
			return nil, nil
		}
		// Handle other potential errors
		return nil, err
	}
	return &snapshot, nil
}

func GetRandomSnapshots(tx *sqlx.Tx) ([]Snapshot, error) {
	query := `
        SELECT * FROM snapshots
        WHERE response_code IS NULL
          AND error IS NULL
        ORDER BY RANDOM()
        LIMIT $1
        FOR UPDATE SKIP LOCKED
    `
	var snapshots []Snapshot
	err := tx.Select(&snapshots, query, config.CONFIG.WorkerBatchSize)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
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

func runWorker(id int, db *sqlx.DB) {
	// Start the transaction
	tx, err := db.Beginx()
	if err != nil {
		logging.LogError("Failed to begin transaction: %w", err)
	}

	defer func() {
		err = tx.Commit()
		if err != nil {
			logging.LogError("[%d] Failed to commit transaction: %w", id, err)
			tx.Rollback()
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
		if InBlacklist(&s) {
			logging.LogDebug("[%d] Ignoring %d/%d blacklisted URL %s", id, i+1, total, s.URL)
		}
		logging.LogInfo("[%d] Starting %d/%d %s", id, i+1, total, s.URL)
		err = workOnSnapshot(id, tx, &s)
		if err != nil {
			logging.LogError("[%d] [%s] Error %w", id, s.URL, err)
			// TODO Remove panic and gracefully handle/log error
			fmt.Printf("Error %s Stack trace:\n%s", err, debug.Stack())
			panic("ERROR encountered")
		}
		if s.Error.Valid {
			logging.LogWarn("[%d] [%s] Error: %v", id, s.URL, fmt.Errorf(s.Error.String))
		}
		logging.LogDebug("[%d] Done %d/%d.", id, i, total)
	}
	logging.LogInfo("[%d] Worker done.", id)
}
