package internal

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTProtected() fiber.Handler {
	return func (c *fiber.Ctx) error {
		auth := c.Get("Authorization")

		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Missing token"})
		}

		token, err := jwt.Parse(auth, func(t *jwt.Token) (any, error) {
			return SecretKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Locals("user_id", claims["user_id"])

		return c.Next()
	}
}