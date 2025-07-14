package main

import (
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/AumSahayata/cloudboxio/db"
	"github.com/AumSahayata/cloudboxio/handlers"
	"github.com/AumSahayata/cloudboxio/internal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

const Version = "1.3.0"

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

	// Setup max upload size
	maxUploadSize, err := strconv.Atoi(os.Getenv("MAX_UPLOAD_SIZE_MB"))
	if err != nil || maxUploadSize <= 0 {
		maxUploadSize = 100
	} 

	// Initiate server
	app := fiber.New(fiber.Config{
		AppName: "CloudBoxIO",
		DisableKeepalive: true,
		BodyLimit:  maxUploadSize << 20,
	})

	// Initiate database
	database, err := db.InitDB()
	if err != nil {
		internal.Error.Fatalln(err)
	}

	defer db.CloseDB(database)
	
	// Apply CORS globally
	app.Use(internal.CORSMiddleware())

	// Rate limiter
	if os.Getenv("ENABLE_RATE_LIMIT") == "true"{
		app.Use(internal.RateLimiterMiddleware())
	}

	// Use default UI for the app
	if os.Getenv("USE_DEFAULT_UI") == "true" {
		// Create a virtual filesystem to server frontend
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

	authHandler := handlers.NewAuthHandler(database, internal.Info, internal.Error)
	fileHandler := handlers.NewFileHandler(database)

	api := app.Group("/api")
	//Public routes
	api.Post("/login", authHandler.Login)
	
	//Protected routes
	api.Use(internal.JWTProtected())
	
	// Files endpoint
	api.Post("/upload:shared?", fileHandler.UploadFile)
	api.Get("/files:shared?", fileHandler.ListFiles)
	api.Get("/file/:fileid", fileHandler.DownloadFile)
	api.Delete("/file/:fileid", fileHandler.DeleteFile)
	
	// User endpoints
	api.Post("/signup", authHandler.SignUp)
	api.Put("/reset-password", authHandler.ResetPassword)
	api.Get("/user-info", authHandler.GetUserInfo)
	api.Get("/users", authHandler.GetUsers)
	api.Delete("/users/:username", authHandler.DeleteUser)

	// Create and hold own TCP listener (not using fiber's listener)
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
}