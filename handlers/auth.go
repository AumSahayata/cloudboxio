package handlers

import (
	"strings"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *fiber.Ctx) error {
	var req models.SignUp
	
	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid Input"})
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username, email, and password are required"})
	}

	// Generate the hash for the password
	hashedpwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Password hashing failed"})
	}

	stmt, err := db.DB.Prepare("INSERT INTO users (id, username, email, password) VALUES (?, ?, ?, ?)")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare statement"})
	}

	_, err = stmt.Exec(uuid.NewString(), req.Username, req.Email, string(hashedpwd))
	if err != nil {
		// Check for SQLite-specific error
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Username or email already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to register user"})	
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message":"User created"})
}

func Login(c *fiber.Ctx) error {
	var req models.Login
	
	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
	}

	// Get user from DB
	row := db.DB.QueryRow(`SELECT id, password FROM users WHERE email = ?`, req.Email)

	var userID, hashedpwd string
	if err := row.Scan(&userID, &hashedpwd); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedpwd), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	// Generate JWT
	token, err := internal.GenerateToken(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to generate token"})
	}

	return c.JSON(fiber.Map{"token":token})
}