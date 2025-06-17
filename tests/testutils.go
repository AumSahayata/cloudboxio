package tests

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func SetupTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
		);
	`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT,
		password TEXT,
		is_admin BOOL
	);
	`)
	if err != nil {
		panic(err)
	}

	return db
}

func SetAdminSetupFlag(db *sql.DB, isDone bool) {
	val := "false"
	if isDone {
		val = "true"
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES ("admin_setup_done", ?)`, val)
	if err != nil {
		log.Printf("SetAdminSetupFlag failed: %v", err)
	}
}