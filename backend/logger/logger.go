package logger

import (
	"fmt"
	"log"
	"os"
)

// Logger holds logging configuration
type Logger struct {
	debug bool
}

// instance is the global logger singleton
var instance *Logger

// init initializes the default logger
func init() {
	// Check if DEBUG environment variable is set
	debugMode := os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"
	instance = &Logger{
		debug: debugMode,
	}
}

// SetDebug sets the debug mode for the logger
func SetDebug(debug bool) {
	instance.debug = debug
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return instance.debug
}

// Info logs an informational message (always shown)
func Info(component, message string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(message, args...)
	log.Printf("[%s] %s", component, formattedMsg)
}

// Debug logs a debug message (only shown if debug mode is enabled)
func Debug(component, message string, args ...interface{}) {
	if !instance.debug {
		return
	}
	formattedMsg := fmt.Sprintf(message, args...)
	log.Printf("[%s] DEBUG: %s", component, formattedMsg)
}

// Error logs an error message (always shown)
func Error(component, message string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(message, args...)
	log.Printf("[%s] ERROR: %s", component, formattedMsg)
}

// ErrorWithErr logs an error with an error object (always shown)
func ErrorWithErr(component, message string, err error) {
	log.Printf("[%s] ERROR: %s: %v", component, message, err)
}

// Panic logs a panic recovery message
func Panic(component, message string, r interface{}) {
	log.Printf("[%s] PANIC: %s: %v", component, message, r)
}

// Raw logs a message without component prefix (for startup messages)
func Raw(message string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(message, args...)
	log.Print(formattedMsg)
}
