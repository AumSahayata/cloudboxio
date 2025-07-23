package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// InitDb initializes the database.
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:data.db?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		log.Fatalln("Failed to open database:", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err = db.Ping(); err != nil {
		log.Fatalln("Database unreachable:", err)
	}

	// createTable is a prepared statement to create metadata table.
	createTable := `CREATE TABLE IF NOT EXISTS metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		filename TEXT,
		size INTEGER,
		path TEXT,
		is_shared BOOLEAN DEFAULT FALSE,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = db.Exec(createTable); err != nil {
		log.Println("Failed to create metadata table:", err)
	}

	// createTable is a prepared statement to create users table.
	createTable = `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		is_admin BOOL DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = db.Exec(createTable); err != nil {
		log.Println("Failed to create users table:", err)
	}

	// createTable is a prepared statement to create settings table.
	createTable = `CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
	);`
	if _, err = db.Exec(createTable); err != nil {
		log.Println("Failed to create settings table:", err)
	}

	// Insert a default 'admin_setup_done' flag if it doesn't exist yet.
	stmt := `INSERT OR IGNORE INTO settings (key, value) VALUES ('admin_setup_done', 'false')`
	if _, err := db.Exec(stmt); err != nil {
		log.Println("Failed to setup initial settings:", err)
	}

	checkAndCreateAdmin(db)

	return db, nil
}

// CloseDB is used to manually close database during graceful shutdown.
func CloseDB(db *sql.DB) {
	if db != nil {
		err := db.Close()
		if err != nil {
			log.Println("Error closing DB:", err)
		} else {
			log.Println("SQLite DB closed.")
		}
	}
}

// createAdmin creates a default admin user.
func createAdmin(db *sql.DB) error {
	// Generate random password
	randomPassword, err := generateRandomPassword(8)
	if err != nil {
		log.Fatalln("Failed to generate password:", err)
		return fmt.Errorf("failed to generate password: %w", err)
	}
	log.Println("Admin password (one-time):", randomPassword)

	// Create temp_admin_credentials.txt file with the generated credentials.
	if err := os.WriteFile("temp_admin_credentials.txt", []byte("CloudBoxIO Temporary Admin Credentials (One-Time Use Only)\n\nUsername: admin\nPassword: "+randomPassword+"\n\nThese credentials are for first-time access only.\nOnce the admin password is reset, this file is deleted automatically."), 0600); err != nil {
		log.Println("failed to write temp admin file:", err)
	}

	log.Println("Admin credentials saved to temp_admin_credentials.txt")

	hashedpwd, err := bcrypt.GenerateFromPassword([]byte(randomPassword), 14)
	if err != nil {
		log.Fatalln("Failed to hash password:", err)
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert a user in users table.
	stmt := `INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`
	if _, err := db.Exec(stmt, uuid.NewString(), "admin", hashedpwd, true); err != nil {
		log.Fatalln("Failed to create admin user:", err)
		return fmt.Errorf("Failed to create admin user: %w", err)
	}

	return nil
}

// checkAndCreateAdmin creates an admin user if it does not exists on first run.
func checkAndCreateAdmin(db *sql.DB) {
	var count int
	// Gets the count of admin users
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&count)
	if err != nil {
		log.Println("Failed to check admin user:", err)
		return
	}
	if count == 0 {
		if err := createAdmin(db); err != nil {
			log.Fatalln("admin creation failed: ", err)
		}
	}
}

// CharapasswordChars is a list of characters to create password from.
const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"

func generateRandomPassword(length int) (string, error) {
	password := make([]byte, length)
	for i := range password {
		// Randomly select characters for password.
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			return "", err
		}
		password[i] = passwordChars[index.Int64()]
	}
	return string(password), nil
}
