package internal

import (
	"io"
	"log"
	"os"
)

var (
	Info  *log.Logger
	Error *log.Logger
)

func InitLogger() {
	_ = os.MkdirAll("logs", os.ModePerm)
	logfile, err := os.OpenFile("logs/server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	log.SetOutput(logfile)

	multi := io.MultiWriter(os.Stdout, logfile)

	Info = log.New(multi, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multi, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}