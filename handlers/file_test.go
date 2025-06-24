package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	// "strings"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
)

func TestUploadFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/upload:shared?", handler.UploadFile)

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
		t.Fatal("Failed to write to test file")
	}
	
	part, err = writer.CreateFormFile("files", filenames[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test2"))
	if err != nil {
		t.Fatal("Failed to write to test file")
	}
	
	writer.Close()
	
	// Create request
	uploadReq := httptest.NewRequest("POST", "/upload?shared=false", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	_, err = ctx.App.Test(uploadReq, -1)
	if err != nil {
		t.Fatal("Protected route request failed:", err)
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
		t.Fatal("Failed to query metadata:", err)
	}
	defer rows.Close()
	found := make(map[string]bool)
	
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to scan metadata: %v", err)
		}
		found[name] = true
	}

	for _, filename := range filenames {
		if !found[filename] {
			t.Fatalf("Expected DB entry for file %s", filename)
		}
	}
}

func TestSharedUploadFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/upload:shared?", handler.UploadFile)

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
		t.Fatal("Failed to write to test file")
	}
	
	part, err = writer.CreateFormFile("files", filenames[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("Hello from test4"))
	if err != nil {
		t.Fatal("Failed to write to test file")
	}
	
	writer.Close()
	
	// Create request
	uploadReq := httptest.NewRequest("POST", "/upload?shared=true", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	_, err = ctx.App.Test(uploadReq, -1)
	if err != nil {
		t.Fatal("Protected route request failed:", err)
	}

	// Check if files were saved
	for _, filename := range filenames {
		expectedPath := filepath.Join(ctx.TempDir, "shared", filename)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Fatalf("expected file to be saved at %s", expectedPath)
		}
	}

	// Validate DB records
	rows, err := ctx.DB.Query("SELECT filename FROM metadata WHERE user_id = ? AND is_shared = ?", "test-id", true)
	if err != nil {
		t.Fatal("Failed to query metadata:", err)
	}
	defer rows.Close()
	found := make(map[string]bool)
	
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to scan metadata: %v", err)
		}
		found[name] = true
	}

	for _, filename := range filenames {
		if !found[filename] {
			t.Fatalf("Expected DB entry for file %s", filename)
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
		path	   string
		uploadedAt string
		isShared   bool
	}{
		{1, "file1.txt", 123, "/path/my-files/file1.txt", "2025-06-01 10:00:00", false},
		{2, "file2.txt", 456, "/path/my-files/file2.txt", "2025-06-02 11:00:00", false},
		{3, "shared.txt", 789, "/path/shared-files/shared.txt", "2025-06-03 12:00:00", true}, // should be excluded
	}

	for _, f := range files {
		_, err := ctx.DB.Exec(`
			INSERT INTO metadata (id, user_id, filename, size, path, is_shared, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			f.id, "test-id", f.filename, f.size, f.path, f.isShared, f.uploadedAt)
		if err != nil {
			t.Fatalf("failed to insert file %s: %v", f.filename, err)
		}
	}

	// Register handler
	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/files:shared?", handler.ListFiles)

	// Make request
	req := httptest.NewRequest("GET", "/files?shared=false", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
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

func TestListSharedFiles(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	// Manually insert test files into the DB
	files := []struct {
		id         int
		filename   string
		size       int
		path	   string
		uploadedAt string
		isShared   bool
	}{
		{1, "file1.txt", 123, "/path/my-files/file1.txt", "2025-06-01 10:00:00", false},  // should be excluded
		{2, "file2.txt", 456, "/path/my-files/file2.txt", "2025-06-02 11:00:00", false},  // should be excluded
		{3, "shared.txt", 789, "/path/shared/shared.txt", "2025-06-03 12:00:00", true},
	}

	for _, f := range files {
		_, err := ctx.DB.Exec(`
			INSERT INTO metadata (id, user_id, filename, size, path, is_shared, uploaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			f.id, "test-id", f.filename, f.size, f.path, f.isShared, f.uploadedAt)
		if err != nil {
			t.Fatalf("failed to insert file %s: %v", f.filename, err)
		}
	}

	// Register handler
	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/files:shared?", handler.ListFiles)

	// Make request
	req := httptest.NewRequest("GET", "/files?shared=true", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
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

	expectedFiles := []string{"shared.txt"}
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
// 		t.Fatal("Failed to write temp file:", err)
// 	}

// 	_, err = ctx.DB.Exec(`INSERT INTO metadata (id, user_id, filename, size, path, is_shared, uploaded_at) VALUES (?,?,?,?,?,?,?)`, 1, "test-id", "test.txt", 1000, tempFilepath, false, "today")
// 	if err != nil {
// 		t.Fatal("Failed to insert temp file record in DB:", err)
// 	}

// 	// Test download
// 	dwnReq := httptest.NewRequest("GET", "/file/1", nil)
// 	dwnReq.Header.Set("Authorization", "Bearer "+ctx.Token)

// 	resp, err := ctx.App.Test(dwnReq, -1)
// 	if err != nil {
// 		t.Fatal("Protected route request failed:", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check status code
// 	if resp.StatusCode != fiber.StatusOK {
// 		t.Fatalf("expected status 200, got %d", resp.StatusCode)
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
		t.Fatal("Failed to write temp file:", err)
	}

	_, err = ctx.DB.Exec(`INSERT INTO metadata (id, user_id, filename, size, path, is_shared, uploaded_at) VALUES (?,?,?,?,?,?,?)`, 1, "test-id", "test.txt", 1000, tempFilepath, false, "today")
	if err != nil {
		t.Fatal("Failed to insert temp file record in DB:", err)
	}


	req := httptest.NewRequest("DELETE", "/file/1", nil)
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatal("Protected route request failed:", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}