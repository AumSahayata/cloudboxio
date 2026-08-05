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
	ID            int
	FileID        string
	FileName      string
	Size          int64
	ExpiresAt     sql.NullString
	DownloadCount int
	MaxDownloads  int
	URL           string
}
