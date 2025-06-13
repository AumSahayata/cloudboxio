package db

import (
	"crypto/rand"
	"database/sql"
	"log"
	"math/big"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// Initialize the database
func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "file:data.db?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		log.Fatalln("Failed to open database:", err)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	if err = DB.Ping(); err != nil {
		log.Fatalln("Database unreachable:", err)
	}
    // remember UNIQUE
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
		log.Println("Failed to create metadata table:", err)
	}

	createTable = `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		is_admin BOOL DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = DB.Exec(createTable); err != nil {
		log.Println("Failed to create users table:", err)
	}
	
	createTable = `CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
	);`
	if _, err = DB.Exec(createTable); err != nil {
		log.Println("Failed to create settings table:", err)
	}

	stmt := `INSERT OR IGNORE INTO settings (key, value) VALUES ('admin_setup_done', 'false')`
	if _, err := DB.Exec(stmt); err != nil {
		log.Println("Failed to setup initial settings:", err)
	}

	checkAndCreateAdmin()
}

func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.Println("Error closing DB:", err)
		} else {
			log.Println("SQLite DB closed.")
		}
	}
}

func createAdmin() {
	randomPassword, err := generateRandomPassword(8)
	if err != nil {
		log.Fatalln("Failed to generate password:", err)
	}
	log.Println("Admin password (one-time):", randomPassword)
	os.WriteFile("temp_admin_credentials.txt", []byte("CloudBoxIO Temporary Admin Credentials (One-Time Use Only)\n\nUsername: admin\nPassword: "+randomPassword+"\n\nThese credentials are for first-time access only.\nOnce the admin password is reset, this file is deleted automatically."), 0600)
	log.Println("Admin credentials saved to temp_admin_credentials.txt")

	hashedpwd, err := bcrypt.GenerateFromPassword([]byte(randomPassword), 14)
	if err != nil {
		log.Fatalln("Failed to hash password:", err)
	}

	stmt := `INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`
	if _, err := DB.Exec(stmt, uuid.NewString(), "admin", hashedpwd, true); err != nil {
		log.Fatalln("Failed to create admin user:", err)
	}
}

func checkAndCreateAdmin() {
    var count int
    err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&count)
    if err != nil {
        log.Println("Failed to check admin user:", err)
        return
    }
    if count == 0 {
        createAdmin()
    }
}

const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"

func generateRandomPassword(length int) (string, error) {
    password := make([]byte, length)
    for i := range password {
        index, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
        if err != nil {
            return "", err
        }
        password[i] = passwordChars[index.Int64()]
    }
    return string(password), nil
}
