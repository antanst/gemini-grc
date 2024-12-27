package main

import (
	"fmt"
	"os"

	"gemini-grc/gemini"
	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for PostgreSQL
	"github.com/jmoiron/sqlx"
)

// Populates the `host` field
func main() {
	db := connectToDB()
	count := 0

	for {
		tx := db.MustBegin()
		query := `
        SELECT * FROM snapshots
        ORDER BY id
        LIMIT 10000 OFFSET $1
    `
		var snapshots []gemini.Snapshot
		err := tx.Select(&snapshots, query, count)
		if err != nil {
			printErrorAndExit(tx, err)
		}
		if len(snapshots) == 0 {
			fmt.Println("Done!")
			return
		}
		for _, s := range snapshots {
			count++
			escaped := gemini.EscapeURL(s.URL.String())
			normalizedGeminiURL, err := gemini.ParseURL(escaped, "")
			if err != nil {
				fmt.Println(s.URL.String())
				fmt.Println(escaped)
				printErrorAndExit(tx, err)
			}
			normalizedURLString := normalizedGeminiURL.String()
			// If URL is already normalized, skip snapshot
			if normalizedURLString == s.URL.String() {
				// fmt.Printf("[%5d] Skipping %d %s\n", count, s.ID, s.URL.String())
				continue
			}
			// If a snapshot already exists with the normalized
			// URL, delete the current snapshot and leave the other.
			var ss []gemini.Snapshot
			err = tx.Select(&ss, "SELECT * FROM snapshots WHERE URL=$1", normalizedURLString)
			if err != nil {
				printErrorAndExit(tx, err)
			}
			if len(ss) > 0 {
				tx.MustExec("DELETE FROM snapshots WHERE id=$1", s.ID)
				fmt.Printf("%d Deleting %d %s\n", count, s.ID, s.URL.String())
				//err = tx.Commit()
				//if err != nil {
				//	printErrorAndExit(tx, err)
				//}
				//return
				continue
			}
			// fmt.Printf("%s =>\n%s\n", s.URL.String(), normalizedURLString)
			// At this point we just update the snapshot,
			// and the normalized URL will be saved.
			fmt.Printf("%d Updating %d %s => %s\n", count, s.ID, s.URL.String(), normalizedURLString)
			// Saves the snapshot with the normalized URL
			tx.MustExec("DELETE FROM snapshots WHERE id=$1", s.ID)
			s.URL = *normalizedGeminiURL
			err = gemini.UpsertSnapshot(0, tx, &s)
			if err != nil {
				printErrorAndExit(tx, err)
			}
			//err = tx.Commit()
			//if err != nil {
			//	printErrorAndExit(tx, err)
			//}
			//return
		}
		err = tx.Commit()
		if err != nil {
			printErrorAndExit(tx, err)
		}
	}
}

func printErrorAndExit(tx *sqlx.Tx, err error) {
	_ = tx.Rollback()
	panic(err)
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
