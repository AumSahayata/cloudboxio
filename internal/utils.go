package internal

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/AumSahayata/cloudboxio/db"
)

func ResolveFileNameConflict(userID, originalName string, isShared bool) (string, error) {
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
			err = db.DB.QueryRow(stmt, finalname).Scan(&exists)
		} else {
			stmt = `SELECT EXISTS(SELECT 1 FROM metadata WHERE filename = ? AND user_id = ? AND is_shared = 0)`
			err = db.DB.QueryRow(stmt, finalname, userID).Scan(&exists)
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
		return "", err
	}

	return cleanedParam, nil
}

// Check for admin setup completion
func IsAdminSetup() bool {
	var adminSetupDone string
	row := db.DB.QueryRow(`SELECT value FROM settings WHERE key = "admin_setup_done"`)
	if err := row.Scan(&adminSetupDone); err != nil {
		return false
	}

	if adminSetupDone == "true" {
		return true
	}

	return false
}

// Change settings
func ChangeSetting(key, newValue string) error {
	_, err := db.DB.Exec(`UPDATE settings SET value = ? WHERE key = ?`, newValue, key)
	if err != nil {
		return err
	}

	return nil
}