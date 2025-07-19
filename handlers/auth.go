package handlers

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB       *sql.DB
	LogINFO  *log.Logger
	LogError *log.Logger
}

func NewAuthHandler(db *sql.DB, infoLogger, errorLogger *log.Logger) *AuthHandler {
	return &AuthHandler{
		DB:       db,
		LogINFO:  infoLogger,
		LogError: errorLogger,
	}
}

func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admin can create users"})
	}
	var req models.SignUp

	// Put the data from the request body into req.
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Input"})
	}

	// Validate required fields.
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password are required"})
	}

	// Generate the hash for the password.
	hashedpwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Password hashing failed"})
	}

	stmt, err := h.DB.Prepare("INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare statement"})
	}

	_, err = stmt.Exec(uuid.NewString(), req.Username, string(hashedpwd), req.IsAdmin)
	if err != nil {
		// Check for SQLite-specific error to check if the username already exists.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}

	adminUsername, err := internal.GetUsernameByID(userID, h.DB)
	if err != nil {
		h.LogINFO.Printf("ADMIN user [%s] created user (%s)", userID, req.Username)
	}

	h.LogINFO.Printf("ADMIN user [%s] created user (%s)", adminUsername, req.Username)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User created"})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedpwd), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	if !internal.IsAdminSetup(h.DB) && !is_admin {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Please login and reset admin password first."})
	}

	// Generate JWT
	token, err := internal.GenerateToken(userID, is_admin, 72)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"token": token})
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
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "All the fields are required"})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 8 characters"})
	}

	// Find user
	row := h.DB.QueryRow("SELECT id, username, password FROM users WHERE id = ?", userID)
	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.Password); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User not found"})
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect current password"})
	}

	// Hash new password
	hashedNew, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	// Update new password
	if _, err := h.DB.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedNew), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
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

	row := h.DB.QueryRow(`SELECT id, username, is_admin FROM users WHERE id = ?`, userID)

	var id string
	var username string
	var isAdmin bool
	if err := row.Scan(&id, &username, &isAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch user data"})
	}

	userData := models.UserInfo{
		ID:       id,
		Username: username,
		IsAdmin:  isAdmin,
	}

	return c.Status(fiber.StatusOK).JSON(userData)
}

func (h *AuthHandler) GetUsers(c *fiber.Ctx) error {
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admin can access users list"})
	}

	rows, err := h.DB.Query(`SELECT id, username, is_admin FROM users`)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Users not found"})
	}

	usersList := []models.UserInfo{}

	for rows.Next() {

		var id string
		var username string
		var isADM bool
		if err := rows.Scan(&id, &username, &isADM); err != nil {
			continue
		}
		usersList = append(usersList, models.UserInfo{
			ID:       id,
			Username: username,
			IsAdmin:  isADM,
		})
	}

	return c.Status(fiber.StatusOK).JSON(usersList)
}

func (h *AuthHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admin can delete users"})
	}

	delID := c.Params("id")
	delID, err := internal.CleanParam(delID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to validate user id"})
	}

	// Check if self delete
	if userID == delID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cannot delete self"})
	}

	// Check if user to delete is admin
	var delUsername string
	var isTargetAdmin bool
	row := h.DB.QueryRow(`SELECT is_admin, username FROM users WHERE id = ?`, delID)
	if err := row.Scan(&isTargetAdmin, &delUsername); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// If deleting an admin, count how many admins are left
	if isTargetAdmin {
		var adminCount int
		err = h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = TRUE`).Scan(&adminCount)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check admin count"})
		}

		if adminCount <= 1 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cannot delete the only remaining admin user"})
		}
	}

	_, err = h.DB.Exec(`DELETE FROM users WHERE id = ?`, delID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete user"})
	}

	adminUsername, err := internal.GetUsernameByID(userID, h.DB)
	if err != nil {
		h.LogINFO.Printf("ADMIN user [%s] deleted user (%s)", userID, delUsername)
	}

	h.LogINFO.Printf("ADMIN user [%s] deleted user (%s)", adminUsername, delUsername)

	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{"message": "User deleted successfully"})
}
