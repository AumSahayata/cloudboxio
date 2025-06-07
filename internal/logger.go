package internal

import (
	"io"
	"log"
	"os"
)

var (
	Info  *log.Logger
	Error *log.Logger
	FileOps  *log.Logger
)

func InitLogger() {
	logFileOps := os.Getenv("LOG_FILE_OPS") == "true"
	logToConsole := os.Getenv("LOG_TO_CONSOLE") == "true"

	_ = os.MkdirAll("logs", os.ModePerm)
	logfile, err := os.OpenFile("logs/server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	
	if err != nil {
		log.Fatalf("error opening server log file: %v", err)
	}

	var output io.Writer = logfile

	if logToConsole {
		output = io.MultiWriter(os.Stdout, logfile)
	}
	
	Info = log.New(output, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(output, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	if logFileOps {
		fileOpsFile, err := os.OpenFile("logs/fileops.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file ops log file: %v", err)
		}

		FileOps = log.New(fileOpsFile, "FILE-OP: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		FileOps = log.New(io.Discard, "", 0)
	}
}