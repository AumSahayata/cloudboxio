package internal

import (
	"bufio"
	"log"
	"os"
	"strings"

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
	envContent := `PORT=3000
LOG_TO_CONSOLE=true
LOG_FILE_OPS=true
USE_DEFAULT_UI=true
FILES_DIR=uploads/
PUBLIC_DIR=public/
ENABLE_RATE_LIMIT=true
RATE_LIMIT_MAX=30
RATE_LIMIT_EXPIRATION_SECOND=30
MAX_UPLOAD_SIZE_MB=100
JWT_EXPIRY_HOURS=24
# Optional TLS (serve HTTPS). Provide paths to a cert and key to enable.
TLS_CERT_FILE=
TLS_KEY_FILE=
`

	_, err = file.WriteString(envContent)
	if err != nil {
		log.Fatalf("Failed to write to .env: %v", err)
	}

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

func UpdateEnvValue(key, value string) bool {
	_, err := os.Stat(".env")
	if err != nil {
		GenerateENV()
	}

	file, err := os.OpenFile(".env", os.O_RDWR, 0644)
	if err != nil {
		Error.Println("Failed to open .env")
		return false
	}

	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	found := false

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			Error.Println(err)
			return false
		}

		line := scanner.Text()

		if strings.HasPrefix(line, key+"=") {
			lines = append(lines, key+"="+value)
			found = true
		} else {
			lines = append(lines, line)
		}
	}

	if !found {
		lines = append(lines, key+"="+value)
	}

	output := strings.Join(lines, "\n") + "\n"

	return os.WriteFile(".env", []byte(output), 0644) == nil
}
