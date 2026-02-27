package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func Init(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("alter0_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, f)

	InfoLogger = log.New(multiWriter, "[INFO] ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

func Info(format string, v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}

// GetCaller returns the caller of the function that called it
func GetCaller() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}
