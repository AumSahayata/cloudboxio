package main

import (
	"github.com/AumSahayata/cloudboxio/handlers"
	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/db"

	"github.com/gofiber/fiber/v2"
)

const Version = "1.0.0"

func main() {
	app := fiber.New()

	// Initiate database
	db.InitDB()

	//Public routes
	app.Post("/signup", handlers.SignUp)
	app.Post("/login", handlers.Login)

	//Protected routes
	app.Use(internal.JWTProtected())

	app.Post("/upload", handlers.UploadFile)
	app.Get("/files", handlers.ListFiles)

	app.Listen(":3000")
}