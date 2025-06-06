package handlers

import (
	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var users = map[string]models.User{}

func SignUp(c *fiber.Ctx) error {
	var req models.User
	
	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid Input"})
	}

	if _, exists := users[req.Email]; exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Email already registered"})
	}

	// Generate the hash for the password
	hashedpwd, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 14)

	newUser := models.User {
		ID:uuid.NewString(),
		Email: req.Email,
		Password: string(hashedpwd),
	}

	users[req.Email] = newUser

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message":"User created"})
}

func Login(c *fiber.Ctx) error {
	var req models.User
	
	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	user, exists := users[req.Email]
	if !exists {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	// Generate JWT
	token, _ := internal.GenerateToken(user.ID)

	return c.JSON(fiber.Map{"token":token})
}