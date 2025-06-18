package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gemini-grc/common/contextlog"
	"gemini-grc/common/snapshot"
	commonUrl "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	"git.antanst.com/antanst/logging"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type DbService interface {
	// Core database methods
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	NewTx(ctx context.Context) (*sqlx.Tx, error)

	// URL methods
	InsertURL(ctx context.Context, tx *sqlx.Tx, url string) error
	CheckAndUpdateNormalizedURL(ctx context.Context, tx *sqlx.Tx, url string, normalizedURL string) error
	DeleteURL(ctx context.Context, tx *sqlx.Tx, url string) error
	MarkURLsAsBeingProcessed(ctx context.Context, tx *sqlx.Tx, urls []string) error
	GetUrlHosts(ctx context.Context, tx *sqlx.Tx) ([]string, error)
	GetRandomUrlsFromHosts(ctx context.Context, hosts []string, limit int, tx *sqlx.Tx) ([]string, error)

	// Snapshot methods
	SaveSnapshot(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error
	OverwriteSnapshot(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error
	UpdateLastCrawled(ctx context.Context, tx *sqlx.Tx, url string) error
	GetLatestSnapshot(ctx context.Context, tx *sqlx.Tx, url string) (*snapshot.Snapshot, error)
	GetSnapshotAtTimestamp(ctx context.Context, tx *sqlx.Tx, url string, timestamp time.Time) (*snapshot.Snapshot, error)
	GetAllSnapshotsForURL(ctx context.Context, tx *sqlx.Tx, url string) ([]*snapshot.Snapshot, error)
	GetSnapshotsByDateRange(ctx context.Context, tx *sqlx.Tx, url string, startTime, endTime time.Time) ([]*snapshot.Snapshot, error)
	IsContentIdentical(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) (bool, error)
}

type DbServiceImpl struct {
	db        *sqlx.DB
	connected bool

	mu sync.Mutex
}

var Database DbServiceImpl

// IsDeadlockError checks if the error is a PostgreSQL deadlock error.
func IsDeadlockError(err error) bool {
	err = errors.Unwrap(err)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "40P01" // PostgreSQL deadlock error code
	}
	return false
}

// Initialize initializes the database with context
func (d *DbServiceImpl) Initialize(ctx context.Context) error {
	// Create a database-specific context
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Initializing database connection")

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return nil
	}

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create a connection pool
	connStr := config.CONFIG.PgURL
	db, err := sqlx.Open("pgx", connStr)
	if err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Unable to connect to database with URL %s: %v", connStr, err)
		return xerrors.NewError(fmt.Errorf("unable to connect to database with URL %s: %w", connStr, err), 0, "", true)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.CONFIG.MaxDbConnections)
	db.SetMaxIdleConns(config.CONFIG.MaxDbConnections / 2)
	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetConnMaxIdleTime(time.Minute * 1)

	// Check if the context is cancelled before proceeding with ping
	if err := ctx.Err(); err != nil {
		return err
	}

	// Use PingContext for context-aware ping
	err = db.PingContext(ctx)
	if err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Unable to ping database: %v", err)
		return xerrors.NewError(fmt.Errorf("unable to ping database: %w", err), 0, "", true)
	}

	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Database connection initialized successfully")
	d.db = db
	d.connected = true
	return nil
}

func (d *DbServiceImpl) Shutdown(ctx context.Context) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Shutting down database connections")
	_, err := d.db.Query("UPDATE urls SET being_processed=false")
	if err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Unable to update urls table: %v", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	err = d.db.Close()
	if err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Error closing database connection: %v", err)
	} else {
		contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Database connection closed successfully")
		d.connected = false
	}
	return err
}

// NewTx creates a new transaction with context
func (d *DbServiceImpl) NewTx(ctx context.Context) (*sqlx.Tx, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Context error before creating transaction: %v", err)
		return nil, err
	}

	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		contextlog.LogErrorWithContext(dbCtx, logging.GetSlogger(), "Failed to create transaction: %v", err)
		return nil, err
	}

	return tx, nil
}

// InsertURL inserts a URL with context
func (d *DbServiceImpl) InsertURL(ctx context.Context, tx *sqlx.Tx, url string) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Inserting URL %s", url)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Context-aware implementation
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

	query := SQL_INSERT_URL
	_, err = tx.NamedExecContext(ctx, query, a)
	if err != nil {
		return xerrors.NewError(fmt.Errorf("cannot insert URL: database error %w URL %s", err, url), 0, "", true)
	}
	return nil
}

// NormalizeURL normalizes a URL with context
func (d *DbServiceImpl) CheckAndUpdateNormalizedURL(ctx context.Context, tx *sqlx.Tx, url string, normalizedURL string) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")

	// Check if URLs are already the same
	if url == normalizedURL {
		return nil
	}

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Updating normalized URL %s -> %s", url, normalizedURL)

	query := SQL_UPDATE_URL
	a := struct {
		Url           string `db:"Url"`
		NormalizedURL string `db:"NormalizedURL"`
	}{
		Url:           url,
		NormalizedURL: normalizedURL,
	}
	_, err := tx.NamedExecContext(ctx, query, a)
	if err != nil {
		return xerrors.NewError(fmt.Errorf("cannot update normalized URL: %w URL %s", err, url), 0, "", true)
	}
	return nil
}

// DeleteURL deletes a URL with context
func (d *DbServiceImpl) DeleteURL(ctx context.Context, tx *sqlx.Tx, url string) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Deleting URL %s", url)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Context-aware implementation
	query := SQL_DELETE_URL
	_, err := tx.ExecContext(ctx, query, url)
	if err != nil {
		return xerrors.NewError(fmt.Errorf("cannot delete URL: database error %w URL %s", err, url), 0, "", true)
	}
	return nil
}

// MarkURLsAsBeingProcessed marks URLs as being processed with context
func (d *DbServiceImpl) MarkURLsAsBeingProcessed(ctx context.Context, tx *sqlx.Tx, urls []string) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")

	// Skip if no URLs provided
	if len(urls) == 0 {
		return nil
	}

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Marking %d URLs as being processed", len(urls))

	// Context-aware implementation
	if len(urls) > 0 {
		// Build a query with multiple parameters instead of using pq.Array
		placeholders := make([]string, len(urls))
		args := make([]interface{}, len(urls))
		for i, url := range urls {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = url
		}
		query := fmt.Sprintf(SQL_MARK_URLS_BEING_PROCESSED, strings.Join(placeholders, ","))
		_, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return xerrors.NewError(fmt.Errorf("cannot mark URLs as being processed: %w", err), 0, "", true)
		}
	}
	return nil
}

// GetUrlHosts gets URL hosts with context
func (d *DbServiceImpl) GetUrlHosts(ctx context.Context, tx *sqlx.Tx) ([]string, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting URL hosts")

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	var hosts []string
	var query string
	if config.CONFIG.GopherEnable {
		query = "SELECT DISTINCT(host) FROM urls WHERE being_processed IS NOT TRUE"
	} else {
		query = "SELECT DISTINCT(host) FROM urls WHERE url like 'gemini://%' AND being_processed IS NOT TRUE"
	}
	err := tx.SelectContext(ctx, &hosts, query)
	if err != nil {
		return nil, xerrors.NewError(err, 0, "", true)
	}
	return hosts, nil
}

// GetRandomUrlsFromHosts gets random URLs from hosts with context
func (d *DbServiceImpl) GetRandomUrlsFromHosts(ctx context.Context, hosts []string, limit int, tx *sqlx.Tx) ([]string, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting random URLs from %d hosts with limit %d", len(hosts), limit)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	var urls []string
	var query string
	for _, host := range hosts {
		var results []string
		if !config.CONFIG.GopherEnable {
			query = "SELECT url FROM urls WHERE host=$1 AND url like 'gemini://%' AND being_processed IS NOT TRUE ORDER BY RANDOM() LIMIT $2"
		} else {
			query = "SELECT url FROM urls WHERE host=$1 AND being_processed IS NOT TRUE ORDER BY RANDOM() LIMIT $2"
		}
		err := tx.SelectContext(ctx, &results, query, host, limit)
		if err != nil {
			return nil, xerrors.NewError(err, 0, "", true)
		}
		urls = append(urls, results...)
	}

	// Check context cancellation before mark operation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Use context-aware method for marking URLs
	err := d.MarkURLsAsBeingProcessed(ctx, tx, urls)
	if err != nil {
		return nil, err
	}
	return urls, nil
}

// SaveSnapshot saves a snapshot with context
func (d *DbServiceImpl) SaveSnapshot(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Saving snapshot for URL %s", s.URL.String())

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Context-aware implementation
	if config.CONFIG.DryRun {
		marshalled, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return xerrors.NewError(fmt.Errorf("JSON serialization error for %v", s), 0, "", true)
		}
		contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Would save snapshot %s", marshalled)
		return nil
	}

	// Check context before expensive operations
	if err := ctx.Err(); err != nil {
		return err
	}

	// Always ensure we have current timestamps
	currentTime := time.Now()
	s.Timestamp = null.TimeFrom(currentTime)
	s.LastCrawled = null.TimeFrom(currentTime)

	// For PostgreSQL, use the global sqlx.NamedQueryContext function
	// The SQL_INSERT_SNAPSHOT already has a RETURNING id clause
	query := SQL_INSERT_SNAPSHOT
	rows, err := sqlx.NamedQueryContext(ctx, tx, query, s)
	if err != nil {
		return xerrors.NewError(fmt.Errorf("cannot save snapshot: %w", err), 0, "", true)
	}
	defer rows.Close()

	// Scan the returned ID
	if rows.Next() {
		err = rows.Scan(&s.ID)
		if err != nil {
			return xerrors.NewError(fmt.Errorf("cannot save snapshot: error scanning returned ID: %w", err), 0, "", true)
		}
	}
	return nil
}

// OverwriteSnapshot overwrites a snapshot with context (maintained for backward compatibility)
func (d *DbServiceImpl) OverwriteSnapshot(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Overwriting snapshot for URL %s", s.URL.String())

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Now simply delegate to SaveSnapshot which is already context-aware
	return d.SaveSnapshot(ctx, tx, s)
}

// UpdateLastCrawled updates the last_crawled timestamp for the most recent snapshot of a URL
func (d *DbServiceImpl) UpdateLastCrawled(ctx context.Context, tx *sqlx.Tx, url string) error {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Updating last_crawled timestamp for URL %s", url)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Update the last_crawled timestamp for the most recent snapshot
	_, err := tx.ExecContext(ctx, SQL_UPDATE_LAST_CRAWLED, url)
	if err != nil {
		return xerrors.NewError(fmt.Errorf("cannot update last_crawled for URL %s: %w", url, err), 0, "", true)
	}

	return nil
}

// GetLatestSnapshot gets the latest snapshot with context
func (d *DbServiceImpl) GetLatestSnapshot(ctx context.Context, tx *sqlx.Tx, url string) (*snapshot.Snapshot, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting latest snapshot for URL %s", url)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	s := &snapshot.Snapshot{}
	err := tx.GetContext(ctx, s, SQL_GET_LATEST_SNAPSHOT, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, xerrors.NewError(fmt.Errorf("no snapshot found for URL %s", url), 0, "", false)
		}
		return nil, xerrors.NewError(fmt.Errorf("cannot get latest snapshot for URL %s: %w", url, err), 0, "", false)
	}
	return s, nil
}

// GetSnapshotAtTimestamp gets a snapshot at a specific timestamp with context
func (d *DbServiceImpl) GetSnapshotAtTimestamp(ctx context.Context, tx *sqlx.Tx, url string, timestamp time.Time) (*snapshot.Snapshot, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting snapshot for URL %s at timestamp %v", url, timestamp)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	s := &snapshot.Snapshot{}
	err := tx.GetContext(ctx, s, SQL_GET_SNAPSHOT_AT_TIMESTAMP, url, timestamp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, xerrors.NewError(fmt.Errorf("no snapshot found for URL %s at or before %v", url, timestamp), 0, "", false)
		}
		return nil, xerrors.NewError(fmt.Errorf("cannot get snapshot for URL %s at timestamp %v: %w", url, timestamp, err), 0, "", false)
	}
	return s, nil
}

// GetAllSnapshotsForURL gets all snapshots for a URL with context
func (d *DbServiceImpl) GetAllSnapshotsForURL(ctx context.Context, tx *sqlx.Tx, url string) ([]*snapshot.Snapshot, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting all snapshots for URL %s", url)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	snapshots := []*snapshot.Snapshot{}
	err := tx.SelectContext(ctx, &snapshots, SQL_GET_ALL_SNAPSHOTS_FOR_URL, url)
	if err != nil {
		return nil, xerrors.NewError(fmt.Errorf("cannot get all snapshots for URL %s: %w", url, err), 0, "", false)
	}
	return snapshots, nil
}

// GetSnapshotsByDateRange gets snapshots by date range with context
func (d *DbServiceImpl) GetSnapshotsByDateRange(ctx context.Context, tx *sqlx.Tx, url string, startTime, endTime time.Time) ([]*snapshot.Snapshot, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Getting snapshots for URL %s in date range %v to %v", url, startTime, endTime)

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Context-aware implementation
	snapshots := []*snapshot.Snapshot{}
	err := tx.SelectContext(ctx, &snapshots, SQL_GET_SNAPSHOTS_BY_DATE_RANGE, url, startTime, endTime)
	if err != nil {
		return nil, xerrors.NewError(fmt.Errorf("cannot get snapshots for URL %s in date range %v to %v: %w",
			url, startTime, endTime, err), 0, "", false)
	}
	return snapshots, nil
}

// IsContentIdentical checks if content is identical with context
func (d *DbServiceImpl) IsContentIdentical(ctx context.Context, tx *sqlx.Tx, s *snapshot.Snapshot) (bool, error) {
	dbCtx := contextutil.ContextWithComponent(ctx, "database")
	contextlog.LogDebugWithContext(dbCtx, logging.GetSlogger(), "Checking if content is identical for URL %s", s.URL.String())

	// Check if the context is cancelled before proceeding
	if err := ctx.Err(); err != nil {
		return false, err
	}

	// Try to get the latest snapshot for this URL
	latestSnapshot := &snapshot.Snapshot{}
	err := tx.GetContext(ctx, latestSnapshot, SQL_GET_LATEST_SNAPSHOT, s.URL.String())
	if err != nil {
		// If there's no snapshot yet, it can't be identical
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, xerrors.NewError(err, 0, "", true)
	}

	// Check context cancellation before potentially expensive comparison
	if err := ctx.Err(); err != nil {
		return false, err
	}

	// Check if the content is identical
	if s.GemText.Valid && latestSnapshot.GemText.Valid {
		return s.GemText.String == latestSnapshot.GemText.String, nil
	} else if s.Data.Valid && latestSnapshot.Data.Valid {
		return bytes.Equal(s.Data.V, latestSnapshot.Data.V), nil
	}

	return false, nil
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
		// Only return error for other types of rollback failures
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to rollback transaction: %v", rollbackErr)
		return xerrors.NewError(fmt.Errorf("failed to rollback transaction: %w", rollbackErr), 0, "", true)
	}
	return nil
}
