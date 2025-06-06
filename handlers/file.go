package handlers

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func UploadFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	//Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File is required"})
	}

	// Create user's folder if not exists
	userDir := filepath.Join("uploads", userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	// Save file to user-specific directory
	savePath := filepath.Join(userDir, file.Filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message":"File uploaded successfully",
		"name":file.Filename,
		"size":file.Size,
	})
}