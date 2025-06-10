package main

import (
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
	app := fiber.New(fiber.Config{
		AppName: "CloudBoxIO",
	})

	// Initiate database
	db.InitDB()
	
	// Apply CORS globally
	app.Use(internal.CORSMiddleware())

	// Use default UI for the app
	if os.Getenv("USE_DEFAULT_UI") == "true"{
		app.Static("/", "./frontend")
	}

	//Public routes
	app.Post("/signup", handlers.SignUp)
	app.Post("/login", handlers.Login)

	//Protected routes
	app.Use(internal.JWTProtected())

	app.Post("/upload/:shared?", handlers.UploadFile)
	app.Get("/my-files", handlers.ListMyFiles)
	app.Get("/shared-files", handlers.ListSharedFiles)
	app.Get("/file/:fileid", handlers.DownloadFile)
	app.Delete("/file/:fileid", handlers.DeleteFile)

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
	
	// Gracefully shutdown the server
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		internal.Error.Printf("Shutdown error: %v", err)
	} else {
		internal.Info.Println("Server shut down gracefully")
	}

	// Close DB
	db.CloseDB()
}