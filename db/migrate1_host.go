package main

import (
	"fmt"
	"gemini-grc/gemini"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
)

func checkIfDone() bool { return true }

// Populates the `host` field
func main() {
	db := connectToDB()

	if checkIfDone() {
		fmt.Println("Migration already applied")
		return
	}

	count := 0
	for {
		// Start the transaction
		tx, err := db.Beginx()
		if err != nil {
			fmt.Println(err)
			return
		}

		query := `
        SELECT * FROM snapshots
        WHERE host IS NULL
        LIMIT 5000
    `
		var snapshots []gemini.Snapshot
		err = tx.Select(&snapshots, query)
		if len(snapshots) == 0 {
			fmt.Println("Done!")
			return
		}
		if err != nil {
			fmt.Println(err)
			err := tx.Rollback()
			if err != nil {
				panic(err)
			}
		}
		for _, s := range snapshots {
			s.Host = s.URL.Hostname
			fmt.Println(count, s.UID, s.URL.Hostname)
			err := gemini.SaveSnapshotToDB(tx, &s)
			if err != nil {
				fmt.Println(err)
				err := tx.Rollback()
				if err != nil {
					panic(err)
				}
			}
			count += 1
		}

		err = tx.Commit()
		if err != nil {
			fmt.Println(err)
			err := tx.Rollback()
			if err != nil {
				panic(err)
			}
		}

	}

}

func connectToDB() *sqlx.DB {
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

	fmt.Println("Connected to database")
	return db
}
