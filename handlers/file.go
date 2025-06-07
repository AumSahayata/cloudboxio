package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
)

func UploadFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	share := c.Query("shared", "false")
	isShare := share == "true"

	//Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File is required"})
	}

	// Create user's folder if not exists
	userDir := filepath.Join("uploads", userID)
	if err := os.MkdirAll(userDir, os.ModePerm); err != nil {
		return err
	}

	filename, err := resolveFileNameConflict(userID, file.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not resolve filename"})
	}

	// Save file to user-specific directory
	savePath := filepath.Join(userDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return err
	}

	// Insert metadata into SQLite DB
	stmt := `INSERT INTO metadata (user_id, filename, size, path, is_shared) VALUES (?, ?, ?, ?, ?);`
	_, err = db.DB.Exec(stmt, userID, filename, file.Size, savePath, isShare)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to save metadata"})
	}

	return c.JSON(fiber.Map{
		"message":"File uploaded successfully",
		"name":filename,
		"size":file.Size,
	})
}

func ListMyFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Query the database for the metadata
	rows, err := db.DB.Query(`SELECT filename, size, uploaded_at FROM metadata WHERE user_id = ? AND is_shared = FALSE`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to query files"})
	}
	defer rows.Close()

	fileList := make([]models.File, 0)

	// Use rows to iterate over the metadata 
	for rows.Next() {
		var filename string
		var size int64
		var uploadedAt string

		if err := rows.Scan(&filename, &size, &uploadedAt); err != nil {
			continue
		}

		fileList = append(fileList, models.File{
			Filename: filename,
			Size: size,
			UploadedAt: uploadedAt,
		})
	}

	return c.JSON(fileList)
}

func ListSharedFiles(c *fiber.Ctx) error {
	// Query the database for the metadata
	rows, err := db.DB.Query(`SELECT filename, size, uploaded_at FROM metadata WHERE is_shared = TRUE`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to query files"})
	}
	defer rows.Close()

	fileList := make([]models.File, 0)

	// Use rows to iterate over the metadata 
	for rows.Next() {
		var filename string
		var size int64
		var uploadedAt string

		if err := rows.Scan(&filename, &size, &uploadedAt); err != nil {
			continue
		}

		fileList = append(fileList, models.File{
			Filename: filename,
			Size: size,
			UploadedAt: uploadedAt,
		})
	}

	return c.JSON(fileList)
}

func DownloadFile(c *fiber.Ctx) error {
	// Get file name from the endpoint parameters using request context
	filename := c.Params("filename")

	// Get sanitized filename
	filename, err := internal.CleanParam(filename)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid filename"})
	}

	// Find the full file path
	var path string

	row := db.DB.QueryRow(`SELECT path FROM metadata WHERE filename = ? LIMIT 1`, filename)
	if err := row.Scan(&path); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found or access denied"})
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error":"File not found"})
	}

	// Send the file as a response
	return c.Download(path)
}

func DeleteFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Get and sanitize filename
	filename := c.Params("filename")
	filename, err := internal.CleanParam(filename)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid filename"})
	}

	// Find the full file path
	var path string

	row := db.DB.QueryRow(`SELECT path FROM metadata WHERE filename = ? AND user_id = ? LIMIT 1`, filename, userID)
	if err := row.Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch file metadata"})
	}

	// Deletes the file from the disk
	if err := os.Remove(path); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete file"})
	}

	// Deletes the metadata of the file
	_, err = db.DB.Exec(`DELETE FROM metadata WHERE filename = ? AND user_id = ?`, filename, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete metadata"})
	}

	return c.JSON(fiber.Map{"message": "File deleted successfully"})
}

func resolveFileNameConflict(userID, originalName string) (string, error) {
	// Split name and extension
	ext := filepath.Ext(originalName)
	base := strings.TrimSuffix(originalName, ext)

	finalname := originalName
	counter := 1

	for {
		var exists bool
		stmt := `SELECT EXISTS(SELECT 1 FROM metadata WHERE filename = ? AND user_id = ?)`

		err := db.DB.QueryRow(stmt, finalname, userID).Scan(&exists)
		if err != nil {
			return "", err
		}

		if !exists {
			break
		}

		finalname = fmt.Sprintf("%s(%d)%s", base, counter, ext)
		counter++
	}

	return finalname, nil
}