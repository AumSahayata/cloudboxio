package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"
	"golang.org/x/crypto/bcrypt"

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
	isPublic := c.QueryBool("public", false)

	fileDir := os.Getenv("FILES_DIR")
	publicDir := os.Getenv("PUBLIC_DIR")

	//Get files from form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File is required"})
	}

	files := form.File["files"]

	// Create public folder if not exists
	dirPath := filepath.Join(fileDir, publicDir)
	if err := os.MkdirAll(dirPath, 0o750); err != nil {
		return fmt.Errorf("failed to create public dir: %w", err)
	}

	if !isPublic {
		// Create user's folder if not exists
		dirPath = filepath.Join(fileDir, userID)
		if err := os.MkdirAll(dirPath, 0o750); err != nil {
			return fmt.Errorf("failed to create user dir: %w", err)
		}
	}

	for _, file := range files {

		// Strip any path from the client provided filename
		safeName := internal.SanitizeFilename(file.Filename)

		filename, err := internal.ResolveFileNameConflict(userID, safeName, isPublic, h.DB)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not resolve filename"})
		}

		// Save file to user-specific directory
		savePath := filepath.Join(dirPath, filename)
		if err := c.SaveFile(file, savePath); err != nil {
			return fmt.Errorf("failed to save the file: %w", err)
		}

		// Insert metadata into SQLite DB
		stmt := `INSERT INTO metadata (user_id, filename, size, path, is_public) VALUES (?, ?, ?, ?, ?);`
		_, err = h.DB.Exec(stmt, userID, filename, file.Size, savePath, isPublic)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save metadata"})
		}

		fileType := "personal"
		if isPublic {
			fileType = "public"
		}

		internal.FileOps.Printf("User [%s] uploaded %s file: %s", userID, fileType, filename)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "File/s uploaded successfully",
	})
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isPublic := c.QueryBool("public", false)
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
		if isPublic {
			stmt, err = h.DB.Prepare(`SELECT md.id, md.filename, md.size, md.uploaded_at, u.username FROM metadata AS md JOIN users AS u ON md.user_id = u.id WHERE md.is_public = TRUE`)
		} else {
			stmt, err = h.DB.Prepare(`SELECT id, filename, size, uploaded_at, "Me" FROM metadata WHERE user_id = ? AND is_public = FALSE`)
		}
	} else {
		if isPublic {
			stmt, err = h.DB.Prepare(`SELECT md.id, md.filename, md.size, md.uploaded_at, u.username FROM metadata AS md JOIN users AS u ON md.user_id = u.id WHERE md.is_public = TRUE AND md.filename LIKE ?`)
		} else {
			stmt, err = h.DB.Prepare(`SELECT id, filename, size, uploaded_at, "Me" FROM metadata WHERE user_id = ? AND is_public = FALSE AND filename LIKE ?`)
		}
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare query"})
	}
	defer stmt.Close()

	// Query the database for the metadata
	if keyword == "" {
		if isPublic {
			rows, err = stmt.Query()
		} else {
			rows, err = stmt.Query(userID)
		}
	} else {
		if isPublic {
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
	userID := c.Locals("user_id").(string)

	// Get file name from the endpoint parameters using request context
	fileID := c.Params("fileid")
	fileID, err := internal.CleanParam(fileID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File ID provided is not proper"})
	}

	// Find the full file path, only if owned by the user or public
	var path string

	row := h.DB.QueryRow(`SELECT path FROM metadata WHERE id = ? AND (user_id = ? OR is_public = TRUE) LIMIT 1`, fileID, userID)
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

	// Find the full file path, public status and owner.
	var isPublic bool
	var path string
	var filename string
	var ownerID string

	row := h.DB.QueryRow(`SELECT filename, is_public, path, user_id FROM metadata WHERE id = ? LIMIT 1`, fileID)
	if err = row.Scan(&filename, &isPublic, &path, &ownerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch file metadata"})
	}

	// Check ownership before removing anything from the disk
	if !isPublic && ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	// Deletes the file from the disk
	if err = os.Remove(path); err != nil {
		internal.FileOps.Println("Error deleting file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete file"})
	}

	// Deletes the metadata of the file
	if _, err = h.DB.Exec(`DELETE FROM metadata WHERE id = ?`, fileID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete metadata"})
	}

	fileType := "personal"
	if isPublic {
		fileType = "public"
	}

	internal.FileOps.Printf("User [%s] deleted %s file: %s", userID, fileType, filename)

	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{"message": "File deleted successfully"})
}


func (h *FileHandler) ShareFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Get file name from the endpoint parameters using request context
	fileID := c.Params("fileid")
	fileID, err := internal.CleanParam(fileID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File ID provided is not proper"})
	}

	var req models.ShareFileMetadata

	// Put the data from the request body into req.
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Input"})
	}

	var expiresAt any

	if req.ExpiresInHours > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)
	}

	var password string

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
		}
		password = string(hashedPassword)
	}

	// Find the full file path
	var path string

	row := h.DB.QueryRow(`SELECT path FROM metadata WHERE id = ?  AND user_id = ?`, fileID, userID)
	if err := row.Scan(&path); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found or access denied"})
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
	}

	token, err := internal.GenerateShareToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate share token"})
	}

	// Insert metadata into SQLite DB
	stmt := `INSERT INTO shares (file_id, token, created_by, expires_at, max_downloads, password_hash) VALUES (?, ?, ?, ?, ?, ?);`
	_, err = h.DB.Exec(stmt, fileID, token, userID, expiresAt, req.MaxDownloads, password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save metadata: " + err.Error()})
	}

	internal.FileOps.Printf("User [%s] shared file: %s", userID, fileID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"url": fmt.Sprintf("%s/api/s/%s", os.Getenv("BASE_URL"), token),
	})
}

func (h *FileHandler) getShareByToken(token string) (*models.Share, error) {
	var share models.Share

	err := h.DB.QueryRow(`
		SELECT
			file_id,
			token,
			expires_at,
			max_downloads,
			download_count,
			password_hash
		FROM shares
		WHERE token = ?
		AND is_active = 1
		LIMIT 1
	`, token).Scan(
		&share.FileID,
		&share.Token,
		&share.ExpiresAt,
		&share.MaxDownloads,
		&share.DownloadCount,
		&share.PasswordHash,
	)

	if err != nil {
		return nil, fiber.ErrNotFound
	}

	if share.ExpiresAt.Valid &&
		time.Now().After(share.ExpiresAt.Time) {
		h.DB.Exec(`UPDATE shares SET is_active = 0 WHERE token = ?`, token)
		return nil, fiber.ErrGone
	}

	if share.MaxDownloads != 0 &&
		share.DownloadCount >= share.MaxDownloads {
		h.DB.Exec(`UPDATE shares SET is_active = 0 WHERE token = ?`, token)
		return nil, fiber.ErrForbidden
	}

	return &share, nil
}

func (h *FileHandler) sendSharedFile(c *fiber.Ctx, share *models.Share) error {

	var path string
	var filename string

	err := h.DB.QueryRow(
		"SELECT filename, path FROM metadata WHERE id = ?",
		share.FileID,
	).Scan(&filename, &path)

	if err != nil {
		return fiber.ErrNotFound
	}

	if _, err := os.Stat(path); err != nil {
		return fiber.ErrNotFound
	}

	_, err = h.DB.Exec(
		`UPDATE shares
		SET download_count = download_count + 1
		WHERE token = ?`,
		share.Token,
	)

	if err != nil {
		return fiber.ErrInternalServerError
	}

	// Send the file as a response
	return c.Status(fiber.StatusOK).Download(path, filename)
}

func (h *FileHandler) ServeSharePage(c *fiber.Ctx) error {
	// Get token from the endpoint parameters using request context
	token := c.Params("token")
	token, err := internal.CleanParam(token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token provided is not proper"})
	}

	if os.Getenv("USE_DEFAULT_UI") == "true" {
		return c.Redirect("/share.html?token=" + token)
	}

	return c.Redirect("/api/share/"+token, fiber.StatusTemporaryRedirect)
}

func (h *FileHandler) GetSharedInfo(c *fiber.Ctx) error {
	// Get token from the endpoint parameters using request context
	token := c.Params("token")
	token, err := internal.CleanParam(token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token provided is not proper"})
	}

	share, err := h.getShareByToken(token)
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Share not found"})
		} else if errors.Is(err, fiber.ErrGone) {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "Share has expired"})
		} else if errors.Is(err, fiber.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Max downloads reached"})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve share"})
	}

	needPassword := share.PasswordHash.Valid && share.PasswordHash.String != ""

	var filename string
	var size int64

	row := h.DB.QueryRow(`SELECT filename, size FROM metadata WHERE id = ? LIMIT 1`, share.FileID)
	if err := row.Scan(&filename, &size); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found or access denied"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"filename":          filename,
		"size":              size,
		"password_required": needPassword,
	})
}

func (h *FileHandler) DownloadSharedFile(c *fiber.Ctx) error {
	token := c.Params("token")
	token, err := internal.CleanParam(token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token provided is not proper"})
	}

	share, err := h.getShareByToken(token)
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Share not found"})
		} else if errors.Is(err, fiber.ErrGone) {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "Share has expired"})
		} else if errors.Is(err, fiber.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Max downloads reached"})
		}
	}

	if share.PasswordHash.Valid && share.PasswordHash.String != "" {
		var req struct {
			Password string `json:"password"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
		}

		if err := bcrypt.CompareHashAndPassword(
			[]byte(share.PasswordHash.String),
			[]byte(req.Password),
		); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid password"})
		}
	}

	return h.sendSharedFile(c, share)
}

func (h *FileHandler) ListMySharedFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	rows, err := h.DB.Query(`
        SELECT s.id, s.file_id, m.filename, m.size, s.expires_at,
        s.download_count, s.max_downloads, s.token
		FROM shares AS s
		JOIN metadata AS m ON s.file_id = m.id
		WHERE s.is_active = TRUE
		AND s.created_by = ?
		AND (s.expires_at IS NULL OR s.expires_at > CURRENT_TIMESTAMP)
		AND (s.max_downloads = 0 OR s.download_count < s.max_downloads)`, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to query files"})
	}
	defer rows.Close()

	files := make([]models.SharedFile, 0)

	for rows.Next() {

		var file models.SharedFile
		var token string

		if err := rows.Scan(&file.ID, &file.FileID, &file.FileName, &file.Size, &file.ExpiresAt, &file.DownloadCount, &file.MaxDownloads, &token); err != nil {
			continue
		}

		file.URL = fmt.Sprintf("%s/api/s/%s", os.Getenv("BASE_URL"), token)

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read rows",
		})
	}

	return c.Status(fiber.StatusOK).JSON(files)
}

func (h *FileHandler) DeactivateShare(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	shareID := c.Params("id")
	shareID, err := internal.CleanParam(shareID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Share ID provided is not proper"})
	}

	res, err := h.DB.Exec(
		`UPDATE shares SET is_active = FALSE WHERE id = ? AND created_by = ?`,
		shareID, userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to deactivate share"})
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to confirm deactivation"})
	}
	if affected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Share not found"})
	}

	internal.FileOps.Printf("User [%s] deactivated share: %s", userID, shareID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Share deactivated"})
}
