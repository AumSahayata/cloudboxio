package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
)

type FileHandler struct {
	DB *sql.DB
}

func NewFileHandler(database *sql.DB) *FileHandler {
	return &FileHandler{DB: database}
}

func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isShared := c.QueryBool("shared", false)

	fileDir := os.Getenv("FILES_DIR")
	sharedDir := os.Getenv("SHARED_DIR")

	//Get files from form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File is required"})
	}

	files := form.File["files"]

	// Create shared folder if not exists
	dirPath := filepath.Join(fileDir, sharedDir)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create shared dir: %w", err)
	}

	if !isShared {
		// Create user's folder if not exists
		dirPath = filepath.Join(fileDir, userID)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create user dir: %w", err)
		}
	}

	for _, file := range files {

		filename, err := internal.ResolveFileNameConflict(userID, file.Filename, isShared, h.DB)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not resolve filename"})
		}

		// Save file to user-specific directory
		savePath := filepath.Join(dirPath, filename)
		if err := c.SaveFile(file, savePath); err != nil {
			return fmt.Errorf("failed to save the file: %w", err)
		}

		// Insert metadata into SQLite DB
		stmt := `INSERT INTO metadata (user_id, filename, size, path, is_shared) VALUES (?, ?, ?, ?, ?);`
		_, err = h.DB.Exec(stmt, userID, filename, file.Size, savePath, isShared)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save metadata"})
		}

		fileType := "personal"
		if isShared {
			fileType = "shared"
		}

		internal.FileOps.Printf("User [%s] uploaded %s file: %s", userID, fileType, filename)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "File/s uploaded successfully",
	})
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isShared := c.QueryBool("shared", false)
	keyword := c.Query("keyword")

	var (
		stmt *sql.Stmt
		rows *sql.Rows
		err  error
	)

	keyword, err = internal.CleanParam(keyword)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Keyword provided is not proper"})
	}

	keyword = "%" + keyword + "%"

	if keyword == "" {
		if isShared {
			stmt, err = h.DB.Prepare(`SELECT md.id, md.filename, md.size, md.uploaded_at, u.username FROM metadata AS md JOIN users AS u ON md.user_id = u.id WHERE md.is_shared = TRUE`)
		} else {
			stmt, err = h.DB.Prepare(`SELECT id, filename, size, uploaded_at, "Me" FROM metadata WHERE user_id = ? AND is_shared = FALSE`)
		}
	} else {
		if isShared {
			stmt, err = h.DB.Prepare(`SELECT md.id, md.filename, md.size, md.uploaded_at, u.username FROM metadata AS md JOIN users AS u ON md.user_id = u.id WHERE md.is_shared = TRUE AND md.filename LIKE ?`)
		} else {
			stmt, err = h.DB.Prepare(`SELECT id, filename, size, uploaded_at, "Me" FROM metadata WHERE user_id = ? AND is_shared = FALSE AND filename LIKE ?`)
		}
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare query"})
	}
	defer stmt.Close()

	// Query the database for the metadata
	if keyword == "" {
		if isShared {
			rows, err = stmt.Query()
		} else {
			rows, err = stmt.Query(userID)
		}
	} else {
		if isShared {
			rows, err = stmt.Query(keyword)
		} else {
			rows, err = stmt.Query(userID, keyword)
		}
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to query files"})
	}
	defer rows.Close()

	fileList := make([]models.File, 0)

	// Use rows to iterate over the metadata
	for rows.Next() {
		var fileID string
		var filename string
		var size int64
		var uploadedAt string
		var uploadedBy string

		if err := rows.Scan(&fileID, &filename, &size, &uploadedAt, &uploadedBy); err != nil {
			continue
		}

		fileList = append(fileList, models.File{
			FileID:     fileID,
			Filename:   filename,
			Size:       size,
			UploadedAt: uploadedAt,
			UploadedBy: uploadedBy,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fileList)
}

func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	// Get file name from the endpoint parameters using request context
	fileID := c.Params("fileid")
	fileID, err := internal.CleanParam(fileID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File ID provided is not proper"})
	}

	// Find the full file path
	var path string

	row := h.DB.QueryRow(`SELECT path FROM metadata WHERE id = ? LIMIT 1`, fileID)
	if err := row.Scan(&path); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found or access denied"})
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
	}

	// Send the file as a response
	return c.Status(fiber.StatusOK).Download(path)
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Get and sanitize filename
	fileID := c.Params("fileid")
	fileID, err := internal.CleanParam(fileID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File ID provided is not proper"})
	}

	// Find the full file path and share status
	var shared bool
	var path string
	var filename string

	row := h.DB.QueryRow(`SELECT filename, is_shared, path FROM metadata WHERE id = ? LIMIT 1`, fileID)
	if err = row.Scan(&filename, &shared, &path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch file metadata"})
	}

	// Deletes the file from the disk
	if err = os.Remove(path); err != nil {
		internal.FileOps.Println("Error deleting file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete file"})
	}

	// Deletes the metadata of the file
	if shared {
		_, err = h.DB.Exec(`DELETE FROM metadata WHERE id = ? AND is_shared = ?`, fileID, true)
	} else {
		_, err = h.DB.Exec(`DELETE FROM metadata WHERE id = ? AND user_id = ? AND is_shared = ?`, fileID, userID, false)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete metadata"})
	}

	fileType := "personal"
	if shared {
		fileType = "shared"
	}

	internal.FileOps.Printf("User [%s] deleted %s file: %s", userID, fileType, filename)

	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{"message": "File deleted successfully"})
}
