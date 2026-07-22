package internal

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenExpiryHours(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int
	}{
		{"default when unset", "", 24},
		{"value from env", "48", 48},
		{"invalid falls back", "abc", 24},
		{"zero falls back", "0", 24},
		{"negative falls back", "-5", 24},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("JWT_EXPIRY_HOURS", c.env)
			if c.env == "" {
				os.Unsetenv("JWT_EXPIRY_HOURS")
			}
			if got := TokenExpiryHours(); got != c.want {
				t.Fatalf("TokenExpiryHours() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestGenerateTokenClaims(t *testing.T) {
	tokenStr, err := GenerateToken("user-42", true, 2)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return SecretKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("failed to parse generated token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		t.Fatal("generated token is not valid")
	}

	if claims["user_id"] != "user-42" {
		t.Fatalf("user_id = %v, want user-42", claims["user_id"])
	}
	if claims["is_admin"] != true {
		t.Fatalf("is_admin = %v, want true", claims["is_admin"])
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		t.Fatal("token has no expiry")
	}
	if remaining := time.Until(exp.Time); remaining < time.Hour || remaining > 2*time.Hour+time.Minute {
		t.Fatalf("unexpected expiry window: %v", remaining)
	}
}

func TestGenerateTokenExpired(t *testing.T) {
	// A token created with a negative lifetime must not pass validation.
	tokenStr, err := GenerateToken("user-42", false, -1)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return SecretKey, nil
	})
	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}
