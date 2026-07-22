package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SecretKey []byte

func InitJWT() {
	SecretKey = []byte(ensureJWTSecret())
}

// TokenExpiryHours returns token lifetime in hours from JWT_EXPIRY_HOURS, defaults to 24.
func TokenExpiryHours() int {
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil && hours > 0 {
			return hours
		}
	}
	return 24
}

func GenerateToken(userID string, isAdmin bool, expTime int) (string, error) {
	// Payload for JWT
	claims := jwt.MapClaims{
		"user_id":  userID,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(time.Hour * time.Duration(expTime)).Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

func ensureJWTSecret() string {
	// Try to get secret from .env
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		return secret
	}

	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)

	if err != nil {
		Error.Printf("failed to generate JWT secret: %v", err)
	}

	secret = hex.EncodeToString(randomBytes)

	// Write jwt secret to .env
	jwtSecret := fmt.Sprintf("JWT_SECRET=%s\n", secret)
	AddToENV(jwtSecret)

	return secret
}
