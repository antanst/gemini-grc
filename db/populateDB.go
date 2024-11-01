package main

import (
	"gemini-grc/uid"
	"time"

	"github.com/jmoiron/sqlx"
)

func PopulateDB(db *sqlx.DB) {
	// Delete all rows in the snapshots table
	db.MustExec("TRUNCATE snapshots;")

	// Prepare the query for inserting a snapshot with uid, url, and timestamp
	query := `INSERT INTO snapshots(uid, url, timestamp)
              VALUES ($1, $2, $3)`

	// Calculate the timestamp for 2 days ago
	timestamp := time.Now().Add(-48 * time.Hour)

	db.MustExec(query, uid.UID(), "gemini://geminiprotocol.net/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://warmedal.se/~antenna", timestamp)
	db.MustExec(query, uid.UID(), "gemini://skyjake.fi/~Cosmos/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://gemini.circumlunar.space/capcom/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://auragem.letz.dev/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://gemplex.space/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://kennedy.gemi.dev/", timestamp)
	db.MustExec(query, uid.UID(), "gemini://tlgs.one/", timestamp)
}
