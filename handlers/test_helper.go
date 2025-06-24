package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type TestContext struct {
	App   *fiber.App
	DB    *sql.DB
	Token string
	TempDir string
}

func SetupTestContext(t *testing.T) *TestContext {
	t.Helper()

	// Setup DB
	database := tests.SetupTestDB()

	// Insert test user
	username := "testuser"
	password := "securepass"
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		t.Fatal("Failed to hash password:", err)
	}
	_, err = database.Exec(`INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`, "test-id", username, hashedPwd, false)
	if err != nil {
		t.Fatalf("Failed to insert user for testing:, %v", err)
	}

	// Setup app
	app := fiber.New()
	handler := NewAuthHandler(database)
	app.Post("/login", handler.Login)

	tests.SetAdminSetupFlag(database, true)

	// Login to get token
	token := loginAndGetToken(t, app, username, password)
	tempDir := setupTestEnv(t)

	t.Cleanup(func() {
		cleanupTestAppAndFiles(t, app, database, tempDir)
	})

	return &TestContext{
		App:   app,
		DB:    database,
		Token: token,
		TempDir: tempDir,
	}
}

// LoginAndGetToken logs in with the given credentials and returns the JWT token
func loginAndGetToken(t *testing.T, app *fiber.App, username, password string) string {
	t.Helper()

	payload := map[string]string{
		"username": username,
		"password": password,
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "aaplication/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Login failed, expected 200 got %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read login response body: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(respBody, &data); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	token := data["token"]
	if token == "" {
		t.Fatal("Login response did not contain a token")
	}

	return token
}

// Create temp dir for testing
func setupTestEnv(t *testing.T) string {
	base := "./test_storage"
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		t.Fatal("Failed to create base dir:", err)
	}

	tempDir, err := os.MkdirTemp(base, "upload*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Set env variables
	os.Setenv("FILES_DIR", tempDir)
	os.Setenv("SHARED_DIR", "shared")
	
    // Initialize FileOps logger to avoid nil panic
    internal.FileOps = log.New(io.Discard, "", 0)

	return tempDir
}

func cleanupTestAppAndFiles(t *testing.T, app *fiber.App, db *sql.DB, tempDir string) {
	t.Helper()
	if err := app.Shutdown(); err != nil {
		t.Fatalf("app shutdown failed: %v", err)
	}
	if err := os.RemoveAll(tempDir); err != nil {
		t.Fatalf("failed to clean temp dir: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close temp db: %v", err)
	}
}
