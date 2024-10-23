package gemini

import (
	"fmt"
	"gemini-grc/logging"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
)

func ConnectToDB() *sqlx.DB {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
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
	db.SetMaxOpenConns(20)
	err = db.Ping()
	if err != nil {
		panic(fmt.Sprintf("Unable to ping database: %v\n", err))
	}

	logging.LogDebug("Connected to database")
	return db
}

func SaveSnapshotToDB(tx *sqlx.Tx, s *Snapshot) error {
	query := `
        INSERT INTO snapshots (uid, url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error)
        VALUES (:uid, :url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error)
        ON CONFLICT (uid) DO UPDATE SET
            url = EXCLUDED.url,
            host = EXCLUDED.host,
            timestamp = EXCLUDED.timestamp,
            mimetype = EXCLUDED.mimetype,
            data = EXCLUDED.data,
            gemtext = EXCLUDED.gemtext,
            links = EXCLUDED.links,
            lang = EXCLUDED.lang,
            response_code = EXCLUDED.response_code,
            error = EXCLUDED.error
    `
	_, err := tx.NamedExec(query, s)
	if err != nil {
		logging.LogError("[%s] [%s] Error upserting snapshot: %w", s.URL, s.MimeType.String, err)
		return fmt.Errorf("DB error: %w", err) // Return the error instead of panicking
	}
	return nil
}

func SaveLinksToDB(tx *sqlx.Tx, snapshots []*Snapshot) error {
	query := `
        INSERT INTO snapshots (uid, url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error)
        VALUES (:uid, :url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error)
        ON CONFLICT (uid) DO NOTHING
    `
	_, err := tx.NamedExec(query, snapshots)
	if err != nil {
		logging.LogError("Error batch inserting snapshots: %w", err)
		return fmt.Errorf("DB error: %w", err)
	}
	return nil
}
