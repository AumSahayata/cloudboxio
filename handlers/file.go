package handlers

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/gofiber/fiber/v2"
)

func UploadFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

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

	// Save file to user-specific directory
	savePath := filepath.Join(userDir, file.Filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return err
	}

	// Insert metadata into SQLite DB
	stmt := `INSERT INTO metadata (user_id, filename, size, path) VALUES (?, ?, ?, ?);`
	_, err = db.DB.Exec(stmt, userID, file.Filename, file.Size, savePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to save metadata"})
	}

	return c.JSON(fiber.Map{
		"message":"File uploaded successfully",
		"name":file.Filename,
		"size":file.Size,
	})
}

func ListFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Query the database for the metadata
	rows, err := db.DB.Query(`SELECT filename, size, uploaded_at FROM metadata WHERE user_id = ?`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to query files"})
	}
	defer rows.Close()

	var fileList []fiber.Map

	// Use rows to iterate over the metadata 
	for rows.Next() {
		var filename string
		var size int64
		var uploadedAt string

		if err := rows.Scan(&filename, &size, &uploadedAt); err != nil {
			continue
		}

		fileList = append(fileList, fiber.Map{
			"filename": filename,
			"size": size,
			"uploaded_at": uploadedAt,
		})
	}

	return c.JSON(fileList)
}

func DownloadFile(c *fiber.Ctx) error {
	// Get file name from the endpoint parameters
	filename := c.Params("filename")

	// Decode %20, %3F etc. to proper characters
	filename, err := url.QueryUnescape(filename)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid filename"})
	}

	// Prevent path traversal (e.g., filename = "../../passwd")
	if strings.Contains(filename, "..") || filepath.IsAbs(filename) {
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