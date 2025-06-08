package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/handlers"
	"github.com/AumSahayata/cloudboxio/internal"

	"github.com/gofiber/fiber/v2"
)

const Version = "1.0.0"

func main() {
	// Ensure .env exists and is loaded
	internal.CheckOrInitEnv()

	// Initiate logger
	internal.InitLogger()
	internal.Info.Println("Starting server...")

	// Initiate JWT
	internal.InitJWT()

	// Initiate server
	app := fiber.New()

	// Initiate database
	db.InitDB()

	// Apply CORS globally
	app.Use(internal.CORSMiddleware())

	//Public routes
	app.Post("/signup", handlers.SignUp)
	app.Post("/login", handlers.Login)

	//Protected routes
	app.Use(internal.JWTProtected())

	app.Post("/upload/:shared?", handlers.UploadFile)
	app.Get("/my-files", handlers.ListMyFiles)
	app.Get("/shared-files", handlers.ListSharedFiles)
	app.Get("/file/:filename", handlers.DownloadFile)
	app.Delete("/file/:filename", handlers.DeleteFile)

	go func() {
		if err := app.Listen(":3000"); err != nil {
			internal.Error.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal (e.g., Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // Block until signal received

	internal.Info.Println("Shutting down server...")

	// Close DB
	db.CloseDB()

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		internal.Error.Printf("Shutdown error: %v", err)
	} else {
		internal.Info.Println("Server shut down gracefully")
	}
}