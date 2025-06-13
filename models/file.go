package models

type File struct {
	FileID string `json:"file_id"`
	Filename string `json:"filename"`
	Size int64 `json:"size"`
	UploadedAt string `json:"uploaded_at"`
	UploadedBy string `json:"uploaded_by"`
}