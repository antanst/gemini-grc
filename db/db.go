package db

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"gemini-grc/common/snapshot"
	commonUrl "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/logging"
	"github.com/antanst/go_errors"
	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func ConnectToDB() (*sqlx.DB, error) {
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
		return nil, go_errors.NewFatalError(fmt.Errorf("unable to connect to database with URL %s: %w", connStr, err))
	}
	// TODO move PG_MAX_OPEN_CONNECTIONS to config env variables
	maxConnections, err := strconv.Atoi(os.Getenv("PG_MAX_OPEN_CONNECTIONS"))
	if err != nil {
		return nil, go_errors.NewFatalError(fmt.Errorf("unable to set DB max connections: %w", err))
	}
	db.SetMaxOpenConns(maxConnections)
	err = db.Ping()
	if err != nil {
		return nil, go_errors.NewFatalError(fmt.Errorf("unable to ping database: %w", err))
	}

	logging.LogDebug("Connected to database")
	return db, nil
}

// IsDeadlockError checks if the error is a PostgreSQL deadlock error.
func IsDeadlockError(err error) bool {
	err = go_errors.Unwrap(err)
	var pqErr *pq.Error
	if go_errors.As(err, &pqErr) {
		return pqErr.Code == "40P01" // PostgreSQL deadlock error code
	}
	return false
}

func GetRandomUrls(tx *sqlx.Tx) ([]string, error) {
	var urls []string
	err := tx.Select(&urls, SQL_SELECT_RANDOM_URLS, config.CONFIG.WorkerBatchSize)
	if err != nil {
		return nil, go_errors.NewFatalError(err)
	}
	return urls, nil
}

func GetRandomUrlsWithBasePath(tx *sqlx.Tx) ([]string, error) {
	SqlQuery := `SELECT url FROM snapshots WHERE url ~ '^[^:]+://[^/]+/?$' ORDER BY RANDOM() LIMIT $1`
	var urls []string
	err := tx.Select(&urls, SqlQuery, config.CONFIG.WorkerBatchSize)
	if err != nil {
		return nil, go_errors.NewFatalError(err)
	}
	return urls, nil
}

func InsertURL(tx *sqlx.Tx, url string) error {
	logging.LogDebug("Inserting URL %s", url)
	query := SQL_INSERT_URL
	normalizedURL, err := commonUrl.ParseURL(url, "", true)
	if err != nil {
		return err
	}
	a := struct {
		Url       string
		Host      string
		Timestamp time.Time
	}{
		Url:       normalizedURL.Full,
		Host:      normalizedURL.Hostname,
		Timestamp: time.Now(),
	}
	_, err = tx.NamedExec(query, a)
	if err != nil {
		return go_errors.NewFatalError(fmt.Errorf("cannot insert URL: database error %w URL %s", err, url))
	}
	return nil
}

func DeleteURL(tx *sqlx.Tx, url string) error {
	logging.LogDebug("Deleting URL %s", url)
	query := SQL_DELETE_URL
	_, err := tx.Exec(query, url)
	if err != nil {
		return go_errors.NewFatalError(fmt.Errorf("cannot delete URL: database error %w URL %s", err, url))
	}
	return nil
}

func OverwriteSnapshot(tx *sqlx.Tx, s *snapshot.Snapshot) (err error) {
	if config.CONFIG.DryRun {
		marshalled, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return go_errors.NewFatalError(fmt.Errorf("JSON serialization error for %v", s))
		}
		logging.LogDebug("Would upsert snapshot %s", marshalled)
		return nil
	}
	query := SQL_UPSERT_SNAPSHOT
	rows, err := tx.NamedQuery(query, s)
	if err != nil {
		return go_errors.NewFatalError(fmt.Errorf("cannot overwrite snapshot: %w", err))
	}
	defer func() {
		_err := rows.Close()
		if err == nil && _err != nil {
			err = go_errors.NewFatalError(fmt.Errorf("cannot overwrite snapshot: error closing rows: %w", err))
		}
	}()
	if rows.Next() {
		var returnedID int
		err = rows.Scan(&returnedID)
		if err != nil {
			return go_errors.NewFatalError(fmt.Errorf("cannot overwrite snapshot: error scanning rows: %w", err))
		}
		s.ID = returnedID
	}
	return nil
}
