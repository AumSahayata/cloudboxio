package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
)

// Signup must reject passwords shorter than 8 characters.
func TestSignup_RejectsWeakPassword(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)

	handler := NewAuthHandler(ctx.DB, ctx.Log, ctx.Log)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Post("/signup", handler.SignUp)

	body, _ := json.Marshal(map[string]any{
		"username": "weakling",
		"password": "123",
		"is_admin": false,
	})
	req := httptest.NewRequest("POST", "/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for weak password, got %d", resp.StatusCode)
	}
}
