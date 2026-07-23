package internal

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func protectedApp() *fiber.App {
	app := fiber.New()
	app.Use(JWTProtected())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func hit(t *testing.T, app *fiber.App, token string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// A token signed with alg=none must be rejected (algorithm-confusion guard).
func TestJWTProtected_RejectsNoneAlg(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id":  "x",
		"is_admin": false,
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}

	if got := hit(t, protectedApp(), signed); got != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 for alg=none token, got %d", got)
	}
}

// A properly signed HS256 token must pass.
func TestJWTProtected_AcceptsHS256(t *testing.T) {
	token, err := GenerateToken("user-1", false, 1)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := hit(t, protectedApp(), token); got != fiber.StatusOK {
		t.Fatalf("expected 200 for valid HS256 token, got %d", got)
	}
}

// Missing token must be rejected.
func TestJWTProtected_RejectsMissingToken(t *testing.T) {
	if got := hit(t, protectedApp(), ""); got == fiber.StatusOK {
		t.Fatal("expected rejection when no token is present")
	}
}
