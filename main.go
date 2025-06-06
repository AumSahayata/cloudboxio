package main

import (
	"github.com/AumSahayata/cloudboxio/handlers"
	"github.com/AumSahayata/cloudboxio/internal"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	//Public routes
	app.Post("/signup", handlers.SignUp)
	app.Post("/login", handlers.Login)

	app.Use(internal.JWTProtected())

	app.Post("/upload", handlers.UploadFile)

	app.Listen(":3000")
}