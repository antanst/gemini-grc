package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"gemini-grc/config"
	"gemini-grc/logging"
	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func ConnectToDB() *sqlx.DB {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", //nolint:nosprintfhostport
		os.Getenv("PG_USER"),
		os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_HOST"),
		os.Getenv("PG_PORT"),
		os.Getenv("PG_DATABASE"),
	)

	// Create a connection pool
	db, err := sqlx.Open("pgx", connStr)
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database with URL %s: %v\n", connStr, err))
	}
	// TODO move PG_MAX_OPEN_CONNECTIONS to config env variables
	maxConnections, err := strconv.Atoi(os.Getenv("PG_MAX_OPEN_CONNECTIONS"))
	if err != nil {
		panic(fmt.Sprintf("Unable to set max DB connections: %s\n", err))
	}
	db.SetMaxOpenConns(maxConnections)
	err = db.Ping()
	if err != nil {
		panic(fmt.Sprintf("Unable to ping database: %v\n", err))
	}

	logging.LogDebug("Connected to database")
	return db
}

// isDeadlockError checks if the error is a PostgreSQL deadlock error
func isDeadlockError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "40P01" // PostgreSQL deadlock error code
	}
	return false
}

func GetSnapshotsToVisit(tx *sqlx.Tx) ([]Snapshot, error) {
	var snapshots []Snapshot
	err := tx.Select(&snapshots, SQL_SELECT_UNVISITED_SNAPSHOTS_UNIQUE_HOSTS, config.CONFIG.WorkerBatchSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabase, err)
	}
	return snapshots, nil
}

func SaveSnapshotIfNew(tx *sqlx.Tx, s *Snapshot) error {
	if config.CONFIG.DryRun {
		marshalled, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			panic(fmt.Sprintf("JSON serialization error for %v", s))
		}
		logging.LogDebug("Would insert (if new) snapshot %s", marshalled)
		return nil
	}
	query := SQL_INSERT_SNAPSHOT_IF_NEW
	_, err := tx.NamedExec(query, s)
	if err != nil {
		return fmt.Errorf("[%s] GeminiError inserting snapshot: %w", s.URL, err)
	}
	return nil
}

func UpsertSnapshot(workedID int, tx *sqlx.Tx, s *Snapshot) (err error) {
	//	if config.CONFIG.DryRun {
	//marshalled, err := json.MarshalIndent(s, "", "  ")
	//if err != nil {
	//	panic(fmt.Sprintf("JSON serialization error for %v", s))
	//}
	//logging.LogDebug("[%d] Would upsert snapshot %s", workedID, marshalled)
	//		return nil
	//	}
	query := SQL_UPSERT_SNAPSHOT
	rows, err := tx.NamedQuery(query, s)
	if err != nil {
		return fmt.Errorf("[%d] %w while upserting snapshot: %w", workedID, ErrDatabase, err)
	}
	defer func() {
		_err := rows.Close()
		if _err != nil {
			err = fmt.Errorf("[%d] %w error closing rows: %w", workedID, ErrDatabase, _err)
		}
	}()
	if rows.Next() {
		var returnedID int
		err = rows.Scan(&returnedID)
		if err != nil {
			return fmt.Errorf("[%d] %w error scanning returned id: %w", workedID, ErrDatabase, err)
		}
		s.ID = returnedID
		// logging.LogDebug("[%d] Upserted snapshot with ID %d", workedID, returnedID)
	}
	return nil
}

func UpdateSnapshot(workedID int, tx *sqlx.Tx, s *Snapshot) (err error) {
	//	if config.CONFIG.DryRun {
	//marshalled, err := json.MarshalIndent(s, "", "  ")
	//if err != nil {
	//	panic(fmt.Sprintf("JSON serialization error for %v", s))
	//}
	//logging.LogDebug("[%d] Would upsert snapshot %s", workedID, marshalled)
	//		return nil
	//	}
	query := SQL_UPDATE_SNAPSHOT
	rows, err := tx.NamedQuery(query, s)
	if err != nil {
		return fmt.Errorf("[%d] %w while updating snapshot: %w", workedID, ErrDatabase, err)
	}
	defer func() {
		_err := rows.Close()
		if _err != nil {
			err = fmt.Errorf("[%d] %w error closing rows: %w", workedID, ErrDatabase, _err)
		}
	}()
	if rows.Next() {
		var returnedID int
		err = rows.Scan(&returnedID)
		if err != nil {
			return fmt.Errorf("[%d] %w error scanning returned id: %w", workedID, ErrDatabase, err)
		}
		s.ID = returnedID
		// logging.LogDebug("[%d] Updated snapshot with ID %d", workedID, returnedID)
	}
	return nil
}

func SaveLinksToDBinBatches(tx *sqlx.Tx, snapshots []*Snapshot) error {
	if config.CONFIG.DryRun {
		return nil
	}
	const batchSize = 5000
	query := SQL_INSERT_SNAPSHOT_IF_NEW
	for i := 0; i < len(snapshots); i += batchSize {
		end := i + batchSize
		if end > len(snapshots) {
			end = len(snapshots)
		}
		batch := snapshots[i:end]
		_, err := tx.NamedExec(query, batch)
		if err != nil {
			return fmt.Errorf("%w: While saving links in batches: %w", ErrDatabase, err)
		}
	}
	return nil
}

func SaveLinksToDB(tx *sqlx.Tx, snapshots []*Snapshot) error {
	if config.CONFIG.DryRun {
		return nil
	}
	query := SQL_INSERT_SNAPSHOT_IF_NEW
	_, err := tx.NamedExec(query, snapshots)
	if err != nil {
		logging.LogError("GeminiError batch inserting snapshots: %w", err)
		return fmt.Errorf("DB error: %w", err)
	}
	return nil
}
