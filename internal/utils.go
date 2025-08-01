package internal

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func ResolveFileNameConflict(userID, originalName string, isShared bool, db *sql.DB) (string, error) {
	// Split name and extension
	ext := filepath.Ext(originalName)
	base := strings.TrimSuffix(originalName, ext)

	finalname := originalName
	counter := 1

	for {
		var exists bool
		var stmt string
		var err error

		// If shared then only check for filename to resolve conflict otherwise also consider user
		if isShared {
			stmt = `SELECT EXISTS(SELECT 1 FROM metadata WHERE filename = ? AND is_shared = 1)`
			err = db.QueryRow(stmt, finalname).Scan(&exists)
		} else {
			stmt = `SELECT EXISTS(SELECT 1 FROM metadata WHERE filename = ? AND user_id = ? AND is_shared = 0)`
			err = db.QueryRow(stmt, finalname, userID).Scan(&exists)
		}

		if err != nil {
			return "", err
		}

		// If filename does not exists break the loop and return new name
		if !exists {
			break
		}

		// Create new file name and increment counter for next time
		finalname = fmt.Sprintf("%s(%d)%s", base, counter, ext)
		counter++
	}

	return finalname, nil
}

func CleanParam(param string) (string, error) {
	// Decode %20, %3F etc. to proper characters
	cleanedParam, err := url.QueryUnescape(param)
	if err != nil {
		return "", err
	}

	// Prevent path traversal (e.g., filename = "../../passwd")
	if strings.Contains(cleanedParam, "..") || filepath.IsAbs(cleanedParam) {
		return "", fmt.Errorf("invalid parameter: potential path traversal")
	}

	return cleanedParam, nil
}

// Check for admin setup completion
func IsAdminSetup(db *sql.DB) bool {
	var adminSetupDone string
	row := db.QueryRow(`SELECT value FROM settings WHERE key = "admin_setup_done"`)
	if err := row.Scan(&adminSetupDone); err != nil {
		return false
	}

	if adminSetupDone == "true" {
		return true
	}

	return false
}

// Change settings
func ChangeSetting(key, newValue string, db *sql.DB) error {
	_, err := db.Exec(`UPDATE settings SET value = ? WHERE key = ?`, newValue, key)
	if err != nil {
		return fmt.Errorf("failed to change settings: %w", err)
	}

	return nil
}

// Get username
func GetUsernameByID(id string, db *sql.DB) (string, error) {
	var username string

	row := db.QueryRow(`SELECT username FROM users WHERE id = ?`, id)

	if err := row.Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		return "", err
	}

	return username, nil
}
