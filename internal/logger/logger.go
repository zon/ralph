package logger

import (
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

// Info logs an informational message (plain string)
func Info(msg string) {
	infoColor.Printf("[INFO] %s\n", msg)
}

// Infof logs an informational message with formatting
func Infof(format string, ctx ...interface{}) {
	infoColor.Printf("[INFO] "+format+"\n", ctx...)
}

// Success logs a success message with checkmark (plain string)
func Success(msg string) {
	successColor.Printf("✓ %s\n", msg)
}

// Successf logs a success message with checkmark and formatting
func Successf(format string, ctx ...interface{}) {
	successColor.Printf("✓ "+format+"\n", ctx...)
}

// Warning logs a warning message (plain string)
func Warning(msg string) {
	warningColor.Printf("[WARNING] %s\n", msg)
}

// Warningf logs a warning message with formatting
func Warningf(format string, ctx ...interface{}) {
	warningColor.Printf("[WARNING] "+format+"\n", ctx...)
}

// Error logs an error message (plain string)
func Error(msg string) {
	errorColor.Printf("[ERROR] %s\n", msg)
}

// Errorf logs an error message with formatting
func Errorf(format string, ctx ...interface{}) {
	errorColor.Printf("[ERROR] "+format+"\n", ctx...)
}

// Verbose logs a verbose/debug message (only if verbose mode is enabled, plain string)
func Verbose(msg string) {
	if !verboseEnabled {
		return
	}
	verboseColor.Printf("[VERBOSE] %s\n", msg)
}

// Verbosef logs a verbose/debug message with formatting (only if verbose mode is enabled)
func Verbosef(format string, ctx ...interface{}) {
	if !verboseEnabled {
		return
	}
	verboseColor.Printf("[VERBOSE] "+format+"\n", ctx...)
}
