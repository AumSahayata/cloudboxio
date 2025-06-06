package internal

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SecretKey = []byte("testSec")

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