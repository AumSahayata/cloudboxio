package main

import (
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/handlers"
	"github.com/AumSahayata/cloudboxio/internal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

const Version = "1.0.0"

//go:embed frontend/*
var embeddedFiles embed.FS

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
		DisableKeepalive: true,
	})

	// Initiate database
	db.InitDB()
	
	// Apply CORS globally
	app.Use(internal.CORSMiddleware())

	// Use default UI for the app
	if os.Getenv("USE_DEFAULT_UI") == "true" {
		subFS, err := fs.Sub(embeddedFiles, "frontend")
		if err != nil {
			internal.Error.Fatalln("Error creating file system for frontend:", err)
		}
		app.Use("/", filesystem.New(filesystem.Config{
			Root: http.FS(subFS),
			Index: "index.html",
			Browse: false,
		}))
		internal.Info.Println("Serving embedded UI at /")
	} else {
		internal.Info.Printf("USE_DEFAULT_UI=false — UI not served")
	}

	//Public routes
	app.Post("/signup", handlers.SignUp)
	app.Post("/login", handlers.Login)

	//Protected routes
	app.Use(internal.JWTProtected())

	// Files endpoint
	app.Post("/upload/:shared?", handlers.UploadFile)
	app.Get("/my-files", handlers.ListMyFiles)
	app.Get("/shared-files", handlers.ListSharedFiles)
	app.Get("/file/:fileid", handlers.DownloadFile)
	app.Delete("/file/:fileid", handlers.DeleteFile)
	
	// User endpoints
	app.Get("/user-info", handlers.GetUserInfo)

	// Create and hold your own TCP listener
    addr := ":" + os.Getenv("PORT")
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        internal.Error.Fatalf("Failed to listen on %s: %v", addr, err)
    }
    internal.Info.Printf("Listening on %s", addr)

	go func() {
		if err := app.Listener(ln); err != nil {
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