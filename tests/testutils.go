package tests

import (
	"database/sql"
	"log"
	"testing"

	_ "modernc.org/sqlite"
)

func SetupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
		);
	`)
	if err != nil {
		t.Fatalf("failed to create settings table: %v", err)
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
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		filename TEXT,
		size INTEGER,
		path TEXT,
		is_shared BOOLEAN DEFAULT FALSE,
		uploaded_at string
		);
	`)
	if err != nil {
		t.Fatalf("failed to create metadata table: %v", err)
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
