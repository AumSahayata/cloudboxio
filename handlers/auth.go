package handlers

import (
	"database/sql"
	"errors"
	"os"
	"strings"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *fiber.Ctx) error {
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin{
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Only admin can create users"})
	}
	var req models.SignUp

	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid Input"})
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password are required"})
	}

	// Generate the hash for the password
	hashedpwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Password hashing failed"})
	}

	stmt, err := db.DB.Prepare("INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare statement"})
	}

	_, err = stmt.Exec(uuid.NewString(), req.Username, string(hashedpwd), req.IsAdmin)
	if err != nil {
		// Check for SQLite-specific error
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Username already exists"})
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
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password are required"})
	}

	// Get user from DB
	row := db.DB.QueryRow(`SELECT id, password, is_admin FROM users WHERE username = ?`, req.Username)

	var userID, hashedpwd string
	var is_admin bool
	if err := row.Scan(&userID, &hashedpwd, &is_admin); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedpwd), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	if !internal.IsAdminSetup() && !is_admin{
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Please login and reset admin password first."})
	}

	// Generate JWT
	token, err := internal.GenerateToken(userID, is_admin, 72)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to generate token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"token":token})
}

func ResetPassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isAdmin := c.Locals("is_admin").(bool)

	var req models.ResetPassword
	// Put the data from the request body into req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Validate required fields
	if req.CurrentPassword == "" || req.NewPassword == ""{
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "All the fields are required"})
	}

	if len(req.NewPassword) < 8 {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 8 characters"})
	}

	// Find user
	row := db.DB.QueryRow("SELECT id, username, password FROM users WHERE id = ?", userID)
    var user models.User
    if err := row.Scan(&user.ID, &user.Username, &user.Password); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"User not found"}) 
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Incorrect current password"})
    }

	// Hash new password
	hashedNew, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to hash password",})
    }

	// Update new password
	if _, err := db.DB.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedNew), userID); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to update password"})
    }

	// Completes the admin setup
	if isAdmin && !internal.IsAdminSetup() {
		if err := internal.ChangeSetting("admin_setup_done", "true"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Password changed, but failed to update system state"})
		}
		if err := os.Remove("temp_admin_credentials.txt"); err != nil {
			internal.Error.Println("Warning: Failed to delete temp admin credentials:", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password reset successful"})
}

func GetUsername(c *fiber.Ctx) error {
	var userID = c.Locals("user_id")

	row := db.DB.QueryRow(`SELECT id, username FROM users WHERE id = ?`, userID)

	var id, username string
	if err := row.Scan(&id, &username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch user data"})
	}

	userData := fiber.Map{
		"ID": id,
		"Username": username,	
	}

	return c.Status(fiber.StatusOK).JSON(userData)
}