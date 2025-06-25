package internal

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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
		c.Locals("is_admin", claims["is_admin"])

		return c.Next()
	}
}


func CORSMiddleware() fiber.Handler {
	// Allowed address 
	var allowedOrigins string = "http://127.0.0.1:" + os.Getenv("PORT")

	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, DELETE, OPTIONS, PUT",
	})
}


func RateLimiterMiddleware() fiber.Handler {
	// Rate limiter configs
	max_limit, err := strconv.Atoi(os.Getenv("RATE_LIMIT_MAX"))
	if err != nil {
		max_limit = 20
	}
	
	rate_limit_exp, err := strconv.Atoi(os.Getenv("RATE_LIMIT_EXPIRATION_SECOND"))
	if err != nil {
		rate_limit_exp = 30
	}

	return limiter.New(limiter.Config{
		Max: max_limit,
		Expiration:  time.Duration(rate_limit_exp)* time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error":"Rate limit exceeded. Try again later."})
		},
	})
}