package db

import (
	"database/sql"

	"github.com/AumSahayata/cloudboxio/internal"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// Initialize the database
func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "file:data.db?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		internal.Error.Fatalln("Failed to open database:", err)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	if err = DB.Ping(); err != nil {
		internal.Error.Fatalln("Database unreachable:", err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		filename TEXT,
		size INTEGER,
		path TEXT,
		is_shared BOOLEAN DEFAULT FALSE,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, filename, is_shared)
	);`
	if _, err = DB.Exec(createTable); err != nil {
		internal.Error.Println("Failed to create metadata table:", err)
	}

	createTable = `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = DB.Exec(createTable); err != nil {
		internal.Error.Println("Failed to create users table:", err)
	}
}

func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			internal.Error.Println("Error closing DB:", err)
		} else {
			internal.Info.Println("SQLite DB closed.")
		}
	}
}
