package handlers

import (
	"database/sql"
	"errors"
	"os"
	"strings"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
    DB *sql.DB
}

func NewAuthHandler(database *sql.DB) *AuthHandler {
    return &AuthHandler{DB: database}
}

func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
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

	stmt, err := h.DB.Prepare("INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)")
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

	internal.Info.Printf("ADMIN user [%s] created new user (%s)", userID, req.Username)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message":"User created"})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
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
	row := h.DB.QueryRow(`SELECT id, password, is_admin FROM users WHERE username = ?`, req.Username)

	var userID, hashedpwd string
	var is_admin bool
	if err := row.Scan(&userID, &hashedpwd, &is_admin); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedpwd), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Invalid credentials"})
	}

	if !internal.IsAdminSetup(h.DB) && !is_admin{
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Please login and reset admin password first."})
	}

	// Generate JWT
	token, err := internal.GenerateToken(userID, is_admin, 72)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to generate token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"token":token})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
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
	row := h.DB.QueryRow("SELECT id, username, password FROM users WHERE id = ?", userID)
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
	if _, err := h.DB.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedNew), userID); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to update password"})
    }

	// Complete admin setup
	if isAdmin && !internal.IsAdminSetup(h.DB) {
		if err := internal.ChangeSetting("admin_setup_done", "true", h.DB); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Password changed, but failed to update system state"})
		}
		// Delete temp_admin_credentials.txt file
		if err := os.Remove("temp_admin_credentials.txt"); err != nil {
			internal.Error.Println("Warning: Failed to delete temp admin credentials:", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password reset successful"})
}

func (h *AuthHandler) GetUserInfo(c *fiber.Ctx) error {
	var userID = c.Locals("user_id")

	row := h.DB.QueryRow(`SELECT username, is_admin FROM users WHERE id = ?`, userID)

	var username string
	var isAdmin bool
	if err := row.Scan(&username, &isAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch user data"})
	}

	userData := models.UserInfo{
		Username: username,	
		IsAdmin: isAdmin,
	}

	return c.Status(fiber.StatusOK).JSON(userData)
}

func (h *AuthHandler) GetUsers(c *fiber.Ctx) error {
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Only admin can access users list"})
	}

	rows, err := h.DB.Query(`SELECT username, is_admin FROM users`)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Users not found"})
	}
	
	usersList := []models.UserInfo{}
	
	for rows.Next() {
		
		var username string
		var isADM bool
		if err := rows.Scan(&username, &isADM); err != nil {
			continue
		}
		usersList = append(usersList, models.UserInfo{
			Username: username,
			IsAdmin: isADM,
		})
	}

	return c.Status(fiber.StatusOK).JSON(usersList)
}

func (h *AuthHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"Only admin can delete users"})
	}

	username := c.Params("username")
	username, err := internal.CleanParam(username)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Failed to validate username"})
	}

	_, err = h.DB.Exec(`DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete user"})
	}

	internal.Info.Printf("ADMIN user [%s] deleted user (%s)", userID, username)
	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{"message": "User deleted successfully"})
}