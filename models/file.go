package models

type File struct {
	Filename string `json:"filename"`
	Size int64 `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}