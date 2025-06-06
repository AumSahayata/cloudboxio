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
	DB, err = sql.Open("sqlite3", "data.db")
	if err != nil {
		internal.Error.Println("Failed to open database:", DB)
	}

	createTable := `CREATE TABLE IF NOT EXISTS metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		filename TEXT,
		size INTEGER,
		path TEXT,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTable)
	if err != nil {
		internal.Error.Println("Failed to create files table:", err)
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
