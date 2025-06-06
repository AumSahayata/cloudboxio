package internal

import (
	"net/url"
	"path/filepath"
	"strings"
)

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