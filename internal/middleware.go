package internal

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-jwt/jwt/v5"
)

func JWTProtected() fiber.Handler {
	return func (c *fiber.Ctx) error {
		// Get JWT
		auth := c.Get("Authorization")

		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Missing token"})
		}

		// Split and get the token
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}
		tokenString := parts[1]

		// Verify the token and validate the signature
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			return SecretKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		// Extract payload from the token
		claims := token.Claims.(jwt.MapClaims)
		// Storeing data in request context
		c.Locals("user_id", claims["user_id"])

		return c.Next()
	}
}

func CORSMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "http://127.0.0.1:3000",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, DELETE, OPTIONS, PUT",
	})
}