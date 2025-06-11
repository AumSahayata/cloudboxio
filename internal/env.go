package internal

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func CheckOrInitEnv() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		GenerateENV()
	}

	// Always load .env so that other packages see updated values
	_ = godotenv.Load()
	log.Println("Loaded .env file")
}

func GenerateENV() bool {
	// Creating a file named .env
	file, err := os.Create(".env")
	if err != nil {
		log.Fatalln("Failed to create .env file:", err)
		return false
	}
	defer file.Close()

	// Writing to .env file
	file.WriteString("PORT=3000\n")
	file.WriteString("LOG_TO_CONSOLE=true\n")
	file.WriteString("LOG_FILE_OPS=true\n")
	file.WriteString("USE_DEFAULT_UI=true\n")
	file.WriteString("FILES_DIR=uploads/\n")
	file.WriteString("SHARED_DIR=shared/\n")

	log.Println("Created .env file")
	return true
}

func AddToENV(content string) bool {
	_, err := os.Stat(".env")
	if err != nil {
		GenerateENV()
	}

	file, err := os.OpenFile(".env", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		Error.Println("Failed to open .env")
		return false
	}

	_, err = file.WriteString(content)
	if err != nil {
		Error.Println("Failed writing to .env")
		return false
	}

	return true
}