package handlers

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/gofiber/fiber/v2"
)


type SettingsHandler struct {
	DB *sql.DB
}

func NewSettingsHandler(database *sql.DB) *SettingsHandler {
	return &SettingsHandler{DB: database}
}

func (h *SettingsHandler) GetBaseURL(c *fiber.Ctx) error {
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admin can update settings"})
	}

	baseURL, err := internal.GetSetting(h.DB, "base_url")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load setting"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"base_url": baseURL})
}

func (h *SettingsHandler) SetBaseURL(c *fiber.Ctx) error {
	isAdmin := c.Locals("is_admin").(bool)

	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admin can update settings"})
	}

	var req struct {
		BaseURL string `json:"base_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	cleaned := strings.TrimSpace(req.BaseURL)
	cleaned = strings.TrimRight(cleaned, "/")

	if cleaned != "" {
		parsed, err := url.Parse(cleaned)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Base URL must be a full URL including scheme, e.g. https://cloudboxio.example.com",
			})
		}
	}

	if err := internal.SetSetting(h.DB, "base_url", cleaned); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save setting"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"base_url": cleaned})
}
