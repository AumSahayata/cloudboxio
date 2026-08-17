package models

import "database/sql"

type File struct {
	FileID     string `json:"file_id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
	UploadedBy string `json:"uploaded_by"`
}

type Share struct {
	ID            int
	FileID        string
	Token         string
	ExpiresAt     sql.NullTime
	MaxDownloads  int
	DownloadCount int
	PasswordHash  sql.NullString
}

type ShareFileMetadata struct {
	ExpiresInHours int    `json:"expires_in_hours"`
	Password       string `json:"password"`
	MaxDownloads   int    `json:"max_downloads"`
}

type SharedFile struct {
	ID               int     `json:"id"`
	FileID           string  `json:"file_id"`
	FileName         string  `json:"filename"`
	Size             int64   `json:"size"`
	ExpiresAt        *string `json:"expires_at"`
	DownloadCount    int     `json:"download_count"`
	MaxDownloads     int     `json:"max_downloads"`
	PasswordRequired bool    `json:"password_required"`
	URL              string  `json:"url"`
}
