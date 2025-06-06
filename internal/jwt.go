package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var SecretKey []byte

func InitJWT() {
	SecretKey = []byte(ensureJWTSecret())
}

func GenerateToken(userID string) (string, error) {
	// Payload for JWT
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp" : time.Now().Add(time.Hour * 72).Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

func ensureJWTSecret() string {
	// Try to load the .env
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		return secret
	}

	// Generate 32 random bytes
	Info.Println("Generating a new JWT secret")

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)

	if err != nil {
		Error.Printf("failed to generate JWT secret: %v", err)
	}

	secret = hex.EncodeToString(randomBytes)

	// Write it to .env
	envContent := fmt.Sprintf("JWT_SECRET = %s\n", secret)
	err = os.WriteFile(".env", []byte(envContent), 0644)
	if err != nil {
		Error.Printf("failed to write .env file: %v", err)
	}

	Info.Println("Generated and stored new JWT_SECRET in .env")
	return secret
}