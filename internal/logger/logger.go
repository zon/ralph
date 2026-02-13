package logger

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	infoColor    = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen)
	warningColor = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	verboseColor = color.New(color.FgWhite, color.Faint)

	verboseEnabled = false
)

// SetVerbose enables or disables verbose logging
func SetVerbose(enabled bool) {
	verboseEnabled = enabled
}

// Info logs an informational message
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	infoColor.Printf("[INFO] %s\n", msg)
}

// Success logs a success message
func Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	successColor.Printf("[SUCCESS] %s\n", msg)
}

// Warning logs a warning message
func Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	warningColor.Printf("[WARNING] %s\n", msg)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	errorColor.Printf("[ERROR] %s\n", msg)
}

// Verbose logs a verbose/debug message (only if verbose mode is enabled)
func Verbose(format string, args ...interface{}) {
	if !verboseEnabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	verboseColor.Printf("[VERBOSE] %s\n", msg)
}
