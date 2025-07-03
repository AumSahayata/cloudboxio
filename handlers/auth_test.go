package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginBlockedBeforeAdminSetup(t *testing.T) {
	db := tests.SetupTestDB()
	defer db.Close()

	tests.SetAdminSetupFlag(db, false)
	voidLogger := log.New(io.Discard, "", 0)

	app := fiber.New()
	handler := NewAuthHandler(db, voidLogger, voidLogger)
	app.Post("/login", handler.Login)

	payload := map[string]string{"username":"admin", "password": "admin"}
	jsonbody, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(jsonbody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db := tests.SetupTestDB()
	defer db.Close()

	tests.SetAdminSetupFlag(db, true)
	voidLogger := log.New(io.Discard, "", 0)

	// Create test user with known credentials
	username := "testuser"
	correctPassword := "correctpass"
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(correctPassword), 14)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	_, err = db.Exec(`INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`, 
		"test-id", username, string(hashedPwd), false)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Setup app + handler
	app := fiber.New()
	handler := NewAuthHandler(db, voidLogger, voidLogger)
	app.Post("/login", handler.Login)

	// Incorrect password
	payload := map[string]string{"username":username, "password": "wrongpass",}
	jsonbody, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(jsonbody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestLoginSuccess(t *testing.T) {
	db := tests.SetupTestDB()
	defer db.Close()

	tests.SetAdminSetupFlag(db, true)
	voidLogger := log.New(io.Discard, "", 0)

	// Create test user with known credentials
	username := "testuser"
	correctPassword := "correctpass"
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(correctPassword), 14)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	_, err = db.Exec(`INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`, 
		"test-id", username, string(hashedPwd), false)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Setup app + handler
	app := fiber.New()
	handler := NewAuthHandler(db, voidLogger, voidLogger)
	app.Post("/login", handler.Login)

	payload := map[string]string{"username":username, "password": correctPassword,}
	jsonbody, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(jsonbody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read response body
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if data["token"] == "" {
		t.Errorf("Expected token in response, got empty string")
	}
}

func TestSignupAsAdmin(t *testing.T) {
	db := tests.SetupTestDB()
	defer db.Close()

	tests.SetAdminSetupFlag(db, true)
	voidLogger := log.New(io.Discard, "", 0)
	
	app := fiber.New()
	handler := NewAuthHandler(db, voidLogger, voidLogger)
	
	// Middleware to inject is_admin = true and user_id
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("is_admin", true)
		c.Locals("user_id", "test-id")
		return c.Next()
	})
	
	app.Post("/signup", handler.SignUp)

	signupData := map[string]any{
		"username": "newuser",
		"password": "strongpass123",
		"is_admin": false,
	}

	jsonbody, err := json.Marshal(signupData)
	if err != nil {
		t.Fatalf("Failed to marshal signup data: %v", err)
	}

	req := httptest.NewRequest("POST", "/signup", bytes.NewReader(jsonbody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestSignupAsNonAdmin(t *testing.T) {
	db := tests.SetupTestDB()
	defer db.Close()

	tests.SetAdminSetupFlag(db, true)
	
	app := fiber.New()
	voidLogger := log.New(io.Discard, "", 0)
	handler := NewAuthHandler(db, voidLogger, voidLogger)
	
	// Middleware to inject is_admin = true
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("is_admin", false)
		c.Locals("user_id", "test-id")
		return c.Next()
	})
	
	app.Post("/signup", handler.SignUp)

	signupData := map[string]any{
		"username": "newuser",
		"password": "strongpass123",
		"is_admin": false,
	}

	jsonbody, err := json.Marshal(signupData)
	if err != nil {
		t.Fatal("Failed to marshal signup data:", err)
	}

	req := httptest.NewRequest("POST", "/signup", bytes.NewReader(jsonbody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestProtectedRouteWithValidToken(t *testing.T) {
	ctx := SetupTestContext(t)

	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewAuthHandler(ctx.DB, ctx.Log, ctx.Log)

	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/user-info", handler.GetUserInfo)

	// --- Access protected route with token ---
	protectedReq := httptest.NewRequest("GET", "/user-info", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	protectedResp, err := ctx.App.Test(protectedReq, -1)
	if err != nil {
		t.Fatal("Protected route request failed:", err)
	}
	
	defer protectedResp.Body.Close()

	if protectedResp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", protectedResp.StatusCode)
	}
}
