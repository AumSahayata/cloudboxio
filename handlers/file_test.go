package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"time"
	"path/filepath"
	// "strings"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func TestUploadFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/upload:public?", handler.UploadFile)

	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filenames := []string{"test1.txt", "test2.txt"}

	part, err := writer.CreateFormFile("files", filenames[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test1"))
	if err != nil {
		t.Fatal("failed to write to test file")
	}

	part, err = writer.CreateFormFile("files", filenames[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test2"))
	if err != nil {
		t.Fatal("failed to write to test file")
	}

	writer.Close()

	// Create request
	uploadReq := httptest.NewRequest("POST", "/upload?public=false", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	_, err = ctx.App.Test(uploadReq, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}

	// Check if files were saved
	for _, filename := range filenames {
		expectedPath := filepath.Join(ctx.TempDir, "test-id", filename)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Fatalf("expected file to be saved at %s", expectedPath)
		}
	}

	// Validate DB records
	rows, err := ctx.DB.Query("SELECT filename FROM metadata WHERE user_id = ?", "test-id")
	if err != nil {
		t.Fatal("failed to query metadata:", err)
	}
	defer rows.Close()
	found := make(map[string]bool)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan metadata: %v", err)
		}
		found[name] = true
	}

	for _, filename := range filenames {
		if !found[filename] {
			t.Fatalf("expected DB entry for file %s", filename)
		}
	}
}

func TestPublicUploadFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/upload:public?", handler.UploadFile)

	// Create multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filenames := []string{"test3.txt", "test4.txt"}

	part, err := writer.CreateFormFile("files", filenames[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test3"))
	if err != nil {
		t.Fatal("failed to write to test file")
	}

	part, err = writer.CreateFormFile("files", filenames[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test4"))
	if err != nil {
		t.Fatal("failed to write to test file")
	}

	writer.Close()

	// Create request
	uploadReq := httptest.NewRequest("POST", "/upload?public=true", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	_, err = ctx.App.Test(uploadReq, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}

	// Check if files were saved
	for _, filename := range filenames {
		expectedPath := filepath.Join(ctx.TempDir, "public", filename)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Fatalf("expected file to be saved at %s", expectedPath)
		}
	}

	// Validate DB records
	rows, err := ctx.DB.Query("SELECT filename FROM metadata WHERE user_id = ? AND is_public = ?", "test-id", true)
	if err != nil {
		t.Fatal("failed to query metadata:", err)
	}
	defer rows.Close()
	found := make(map[string]bool)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan metadata: %v", err)
		}
		found[name] = true
	}

	for _, filename := range filenames {
		if !found[filename] {
			t.Fatalf("expected DB entry for file %s", filename)
		}
	}
}

func TestListMyFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	// Manually insert test files into the DB
	files := []struct {
		id         int
		filename   string
		size       int
		path       string
		uploadedAt string
		isPublic   bool
	}{
		{1, "file1.txt", 123, "/path/my-files/file1.txt", "2025-06-01 10:00:00", false},
		{2, "file2.txt", 456, "/path/my-files/file2.txt", "2025-06-02 11:00:00", false},
		{3, "public.txt", 789, "/path/public-files/public.txt", "2025-06-03 12:00:00", true}, // should be excluded
	}

	for _, f := range files {
		_, err := ctx.DB.Exec(`
			INSERT INTO metadata (id, user_id, filename, size, path, is_public, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			f.id, "test-id", f.filename, f.size, f.path, f.isPublic, f.uploadedAt)
		if err != nil {
			t.Fatalf("failed to insert file %s: %v", f.filename, err)
		}
	}

	// Register handler
	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/files:public?", handler.ListFiles)

	// Make request
	req := httptest.NewRequest("GET", "/files?public=false", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// Parse response
	var result []models.File
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	expectedFiles := []string{"file1.txt", "file2.txt"}
	receivedFiles := make(map[string]bool)

	// Check expected files
	for _, file := range result {
		receivedFiles[file.Filename] = true
	}

	if len(result) != len(expectedFiles) {
		t.Fatalf("expected %d files, got %d", len(expectedFiles), len(result))
	}

	for _, f := range expectedFiles {
		if !receivedFiles[f] {
			t.Errorf("unexpected file in result: %s", f)
		}
	}
}

func TestListPublicFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	// Manually insert test files into the DB
	files := []struct {
		id         int
		filename   string
		size       int
		path       string
		uploadedAt string
		isPublic   bool
	}{
		{1, "file1.txt", 123, "/path/my-files/file1.txt", "2025-06-01 10:00:00", false}, // should be excluded
		{2, "file2.txt", 456, "/path/my-files/file2.txt", "2025-06-02 11:00:00", false}, // should be excluded
		{3, "public.txt", 789, "/path/public-files/public.txt", "2025-06-03 12:00:00", true},
	}

	for _, f := range files {
		_, err := ctx.DB.Exec(`
			INSERT INTO metadata (id, user_id, filename, size, path, is_public, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			f.id, "test-id", f.filename, f.size, f.path, f.isPublic, f.uploadedAt)
		if err != nil {
			t.Fatalf("failed to insert file %s: %v", f.filename, err)
		}
	}

	// Register handler
	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/files:public?", handler.ListFiles)

	// Make request
	req := httptest.NewRequest("GET", "/files?public=true", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// Parse response
	var result []models.File
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	expectedFiles := []string{"public.txt"}
	receivedFiles := make(map[string]bool)

	// Check expected files
	for _, file := range result {
		receivedFiles[file.Filename] = true
	}

	if len(result) != len(expectedFiles) {
		t.Fatalf("expected %d files, got %d", len(expectedFiles), len(result))
	}

	for _, f := range expectedFiles {
		if !receivedFiles[f] {
			t.Errorf("unexpected file in result: %s", f)
		}
	}
}

// Does not exits the test_storage (temp dir for testing) hence throws an error

// func TestDownloadFile(t *testing.T) {
// 	ctx := SetupTestContext(t)
// 	tests.SetAdminSetupFlag(ctx.DB, true)

// 	handler := NewFileHandler(ctx.DB)
// 	ctx.App.Use(internal.JWTProtected())
// 	ctx.App.Get("/file/:fileid", handler.DownloadFile)

// 	// Create a dummy file
// 	tempFilepath := filepath.Join(ctx.TempDir, "test.txt")
// 	fileContent := []byte("This is a test file.")
// 	err := os.WriteFile(tempFilepath, fileContent, os.ModePerm)
// 	if err != nil {
// 		t.Fatal("failed to write temp file:", err)
// 	}

// 	_, err = ctx.DB.Exec(`INSERT INTO metadata (id, user_id, filename, size, path, is_public, uploaded_at) VALUES (?,?,?,?,?,?,?)`, 1, "test-id", "test.txt", 1000, tempFilepath, false, "today")
// 	if err != nil {
// 		t.Fatal("failed to insert temp file record in DB:", err)
// 	}

// 	// Test download
// 	dwnReq := httptest.NewRequest("GET", "/file/1", nil)
// 	dwnReq.Header.Set("Authorization", "Bearer "+ctx.Token)

// 	resp, err := ctx.App.Test(dwnReq, -1)
// 	if err != nil {
// 		t.Fatal("request failed:", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check status code
// 	if resp.StatusCode != fiber.StatusOK {
// 		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
// 	}

// 	// Check header
// 	contentDisposition := resp.Header.Get("Content-Disposition")
// 	if !strings.HasPrefix(contentDisposition, "attachment;") {
// 		t.Fatalf("expected Content-Disposition to be attachment, got: %s", contentDisposition)
// 	}

// 	// Check file content
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Fatalf("failed to read response body: %v", err)
// 	}

// 	if !bytes.Equal(body, fileContent) {
// 		t.Fatalf("expected body does not match, got %q", body)
// 	}
// }

func TestDeleteFile(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Delete("/file/:fileid", handler.DeleteFile)

	// Create a dummy file
	tempFilepath := filepath.Join(ctx.TempDir, "test.txt")
	fileContent := []byte("This is a test file.")
	err := os.WriteFile(tempFilepath, fileContent, os.ModePerm)
	if err != nil {
		t.Fatal("failed to write temp file:", err)
	}

	_, err = ctx.DB.Exec(`INSERT INTO metadata (id, user_id, filename, size, path, is_public, uploaded_at) VALUES (?,?,?,?,?,?,?)`, 1, "test-id", "test.txt", 1000, tempFilepath, false, "today")
	if err != nil {
		t.Fatal("failed to insert temp file record in DB:", err)
	}

	req := httptest.NewRequest("DELETE", "/file/1", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected status %d, got %d", fiber.StatusNoContent, resp.StatusCode)
	}
}

func TestShareFile(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	_, err := ctx.DB.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)`,
		"base_url",
		"http://localhost:3000",
	)
	if err != nil {
		t.Fatal("failed to insert base URL:", err)
	}

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/file/:fileid/share", handler.ShareFile)

	// Create a real file because ShareFile checks the filesystem.
	filePath := filepath.Join(ctx.TempDir, "share-me.txt")
	fileContent := []byte("file content for sharing")

	if err := os.WriteFile(filePath, fileContent, os.ModePerm); err != nil {
		t.Fatal("failed to create test file:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"share-me.txt",
		len(fileContent),
		filePath,
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	payload := map[string]interface{}{
		"expires_in_hours": 24,
		"max_downloads":    5,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal("failed to marshal request:", err)
	}

	req := httptest.NewRequest(
		"POST",
		"/file/1/share",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected status %d, got %d", fiber.StatusCreated, resp.StatusCode)
	}

	var result map[string]string
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("failed to read response:", err)
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatal("failed to parse response:", err)
	}

	if result["url"] == "" {
		t.Fatal("expected share URL in response")
	}

	if !bytes.HasPrefix(
		[]byte(result["url"]),
		[]byte("http://localhost:3000/api/s/"),
	) {
		t.Fatalf("unexpected share URL: %s", result["url"])
	}

	// Verify DB state.
	var (
		fileID       int
		createdBy     string
		maxDownloads  int
		downloadCount int
		token        string
		isActive     bool
	)

	err = ctx.DB.QueryRow(`
		SELECT file_id, created_by, max_downloads,
		       download_count, token, is_active
		FROM shares
		WHERE file_id = ?`,
		1,
	).Scan(
		&fileID,
		&createdBy,
		&maxDownloads,
		&downloadCount,
		&token,
		&isActive,
	)
	if err != nil {
		t.Fatal("failed to query created share:", err)
	}

	if fileID != 1 {
		t.Errorf("expected file ID 1, got %d", fileID)
	}

	if createdBy != "test-id" {
		t.Errorf("expected created_by test-id, got %s", createdBy)
	}

	if maxDownloads != 5 {
		t.Errorf("expected max downloads 5, got %d", maxDownloads)
	}

	if downloadCount != 0 {
		t.Errorf("expected initial download count 0, got %d", downloadCount)
	}

	if token == "" {
		t.Fatal("expected generated token")
	}

	if !isActive {
		t.Fatal("expected share to be active")
	}
}

func TestShareFileAccessDenied(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	_, err := ctx.DB.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)`,
		"base_url",
		"http://localhost:3000",
	)
	if err != nil {
		t.Fatal("failed to insert base URL:", err)
	}

	filePath := filepath.Join(ctx.TempDir, "other-user.txt")
	if err := os.WriteFile(filePath, []byte("private"), os.ModePerm); err != nil {
		t.Fatal("failed to create test file:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"other-user",
		"other-user.txt",
		7,
		filePath,
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/file/:fileid/share", handler.ShareFile)

	req := httptest.NewRequest(
		"POST",
		"/file/1/share",
		bytes.NewReader([]byte(`{}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			fiber.StatusNotFound,
			resp.StatusCode,
		)
	}
}

func TestGetSharedInfo(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Get("/api/share/:token", handler.GetSharedInfo)

	_, err := ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"document.pdf",
		12345,
		"/does/not/matter/document.pdf",
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, expires_at,
			 max_downloads, download_count, password_hash, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-share-token",
		"test-id",
		nil,
		0,
		0,
		nil,
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	req := httptest.NewRequest(
		"GET",
		"/api/share/test-share-token",
		nil,
	)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var result struct {
		Filename        string `json:"filename"`
		Size            int64  `json:"size"`
		PasswordRequired bool  `json:"password_required"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("failed to read response:", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal("failed to parse response:", err)
	}

	if result.Filename != "document.pdf" {
		t.Errorf("expected document.pdf, got %s", result.Filename)
	}

	if result.Size != 12345 {
		t.Errorf("expected size 12345, got %d", result.Size)
	}

	if result.PasswordRequired {
		t.Error("expected password not to be required")
	}
}

func TestGetSharedInfoExpired(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Get("/api/share/:token", handler.GetSharedInfo)

	_, err := ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"expired.txt",
		100,
		"/expired.txt",
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	expired := time.Now().Add(-1 * time.Hour)

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, expires_at,
			 max_downloads, download_count, password_hash, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		1,
		"expired-token",
		"test-id",
		expired,
		0,
		0,
		nil,
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	req := httptest.NewRequest(
		"GET",
		"/api/share/expired-token",
		nil,
	)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusGone {
		t.Fatalf(
			"expected status %d, got %d",
			fiber.StatusGone,
			resp.StatusCode,
		)
	}

	var isActive bool

	err = ctx.DB.QueryRow(
		`SELECT is_active FROM shares WHERE token = ?`,
		"expired-token",
	).Scan(&isActive)
	if err != nil {
		t.Fatal("failed to query share:", err)
	}

	if isActive {
		t.Error("expected expired share to be deactivated")
	}
}

func TestDownloadSharedFileWithPassword(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Post("/api/s/:token", handler.DownloadSharedFile)

	filePath := filepath.Join(ctx.TempDir, "secret.txt")
	fileContent := []byte("this is secret content")

	if err := os.WriteFile(filePath, fileContent, os.ModePerm); err != nil {
		t.Fatal("failed to create test file:", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.MinCost,
	)
	if err != nil {
		t.Fatal("failed to hash password:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"secret.txt",
		len(fileContent),
		filePath,
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, expires_at,
			 max_downloads, download_count, password_hash, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		1,
		"password-token",
		"test-id",
		nil,
		0,
		0,
		string(hashedPassword),
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	body := bytes.NewReader([]byte(`{"password":"correct-password"}`))

	req := httptest.NewRequest(
		"POST",
		"/api/s/password-token",
		body,
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("failed to read response:", err)
	}

	if !bytes.Equal(responseBody, fileContent) {
		t.Fatalf(
			"expected file content %q, got %q",
			fileContent,
			responseBody,
		)
	}

	var downloadCount int

	err = ctx.DB.QueryRow(
		`SELECT download_count FROM shares WHERE token = ?`,
		"password-token",
	).Scan(&downloadCount)
	if err != nil {
		t.Fatal("failed to query download count:", err)
	}

	if downloadCount != 1 {
		t.Fatalf("expected download count 1, got %d", downloadCount)
	}
}

func TestDownloadSharedFileInvalidPassword(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Post("/api/s/:token", handler.DownloadSharedFile)

	filePath := filepath.Join(ctx.TempDir, "secret.txt")

	if err := os.WriteFile(
		filePath,
		[]byte("secret"),
		os.ModePerm,
	); err != nil {
		t.Fatal("failed to create test file:", err)
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.MinCost,
	)
	if err != nil {
		t.Fatal("failed to hash password:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"secret.txt",
		6,
		filePath,
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, password_hash, is_active)
		VALUES (?, ?, ?, ?, ?)`,
		1,
		"protected-token",
		"test-id",
		string(hash),
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	req := httptest.NewRequest(
		"POST",
		"/api/s/protected-token",
		bytes.NewReader([]byte(`{"password":"wrong-password"}`)),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			fiber.StatusUnauthorized,
			resp.StatusCode,
		)
	}
}

func TestDownloadSharedFileMaxDownloads(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Get("/api/s/:token", handler.DownloadSharedFile)

	filePath := filepath.Join(ctx.TempDir, "limited.txt")

	if err := os.WriteFile(
		filePath,
		[]byte("limited"),
		os.ModePerm,
	); err != nil {
		t.Fatal("failed to create test file:", err)
	}

	_, err := ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"limited.txt",
		7,
		filePath,
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, max_downloads,
			 download_count, is_active)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1,
		"limited-token",
		"test-id",
		1,
		1,
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	req := httptest.NewRequest(
		"GET",
		"/api/s/limited-token",
		nil,
	)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			fiber.StatusForbidden,
			resp.StatusCode,
		)
	}

	var isActive bool

	err = ctx.DB.QueryRow(
		`SELECT is_active FROM shares WHERE token = ?`,
		"limited-token",
	).Scan(&isActive)
	if err != nil {
		t.Fatal("failed to query share:", err)
	}

	if isActive {
		t.Error("expected share to be deactivated after max downloads")
	}
}

func TestListMySharedFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	_, err := ctx.DB.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)`,
		"base_url",
		"http://localhost:3000",
	)
	if err != nil {
		t.Fatal("failed to insert base URL:", err)
	}

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/shares", handler.ListMySharedFiles)

	files := []struct {
		id       int
		filename string
		size     int
	}{
		{1, "file1.txt", 100},
		{2, "file2.pdf", 200},
		{3, "expired.txt", 300},
	}

	for _, file := range files {
		path := filepath.Join(ctx.TempDir, file.filename)

		if err := os.WriteFile(path, []byte("test"), os.ModePerm); err != nil {
			t.Fatal("failed to create test file:", err)
		}

		_, err := ctx.DB.Exec(`
			INSERT INTO metadata
				(id, user_id, filename, size, path, is_public, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			file.id,
			"test-id",
			file.filename,
			file.size,
			path,
			false,
			"today",
		)
		if err != nil {
			t.Fatal("failed to insert metadata:", err)
		}
	}

	// Valid share.
	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, max_downloads,
			 download_count, is_active)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1,
		"token-1",
		"test-id",
		0,
		0,
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share 1:", err)
	}

	// Valid share with password.
	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, max_downloads,
			 download_count, password_hash, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		2,
		"token-2",
		"test-id",
		5,
		1,
		"hashed-password",
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share 2:", err)
	}

	// Expired share - should not appear.
	expired := time.Now().Add(-1 * time.Hour)

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(file_id, token, created_by, expires_at,
			 max_downloads, download_count, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		3,
		"expired-token",
		"test-id",
		expired,
		0,
		0,
		true,
	)
	if err != nil {
		t.Fatal("failed to insert expired share:", err)
	}

	req := httptest.NewRequest("GET", "/shares", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var result []models.SharedFile

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("failed to read response:", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal("failed to parse response:", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 active shares, got %d", len(result))
	}

	found := make(map[string]models.SharedFile)

	for _, file := range result {
		found[file.FileName] = file
	}

	if _, ok := found["expired.txt"]; ok {
		t.Error("expired share should not be returned")
	}

	if found["file1.txt"].URL != "http://localhost:3000/api/s/token-1" {
		t.Errorf(
			"unexpected URL for file1: %s",
			found["file1.txt"].URL,
		)
	}

	if found["file2.pdf"].DownloadCount != 1 {
		t.Errorf(
			"expected download count 1, got %d",
			found["file2.pdf"].DownloadCount,
		)
	}

	if !found["file2.pdf"].PasswordRequired {
		t.Error("expected file2.pdf to require password")
	}
}

func TestDeactivateShare(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Delete("/shares/:id", handler.DeactivateShare)

	_, err := ctx.DB.Exec(`
		INSERT INTO metadata
			(id, user_id, filename, size, path, is_public, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1,
		"test-id",
		"file.txt",
		100,
		"/file.txt",
		false,
		"today",
	)
	if err != nil {
		t.Fatal("failed to insert metadata:", err)
	}

	_, err = ctx.DB.Exec(`
		INSERT INTO shares
			(id, file_id, token, created_by, is_active)
		VALUES (?, ?, ?, ?, ?)`,
		1,
		1,
		"deactivate-token",
		"test-id",
		true,
	)
	if err != nil {
		t.Fatal("failed to insert share:", err)
	}

	req := httptest.NewRequest(
		"DELETE",
		"/shares/1",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var isActive bool

	err = ctx.DB.QueryRow(
		`SELECT is_active FROM shares WHERE id = ?`,
		1,
	).Scan(&isActive)
	if err != nil {
		t.Fatal("failed to query share:", err)
	}

	if isActive {
		t.Error("expected share to be inactive")
	}
}