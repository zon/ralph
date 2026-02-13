package logger

import (
	"testing"
)

// Note: Testing logger output directly is challenging because the color package
// writes to the underlying file descriptors. Instead, we test the behavior
// and state management.

func TestInfo(t *testing.T) {
	// Test that Info doesn't panic and accepts format strings
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Info panicked: %v", r)
		}
	}()
	Info("test message")
	Info("formatted: %s, %d", "test", 42)
}

func TestSuccess(t *testing.T) {
	// Test that Success doesn't panic and accepts format strings
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Success panicked: %v", r)
		}
	}()
	Success("operation completed")
	Success("completed: %d items", 5)
}

func TestWarning(t *testing.T) {
	// Test that Warning doesn't panic and accepts format strings
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Warning panicked: %v", r)
		}
	}()
	Warning("potential issue")
	Warning("warning: %s", "something")
}

func TestError(t *testing.T) {
	// Test that Error doesn't panic and accepts format strings
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Error panicked: %v", r)
		}
	}()
	Error("something failed")
	Error("error: %v", "details")
}

func TestVerboseWhenDisabled(t *testing.T) {
	SetVerbose(false)

	// Test that Verbose doesn't panic when disabled
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Verbose panicked when disabled: %v", r)
		}
	}()

	Verbose("debug info")
	Verbose("debug: %s", "test")
}

func TestVerboseWhenEnabled(t *testing.T) {
	SetVerbose(true)
	defer SetVerbose(false) // Reset after test

	// Test that Verbose doesn't panic when enabled
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Verbose panicked when enabled: %v", r)
		}
	}()

	Verbose("debug info")
	Verbose("debug: %d", 123)
}

func TestSetVerbose(t *testing.T) {
	// Test enabling verbose mode
	SetVerbose(true)
	if !verboseEnabled {
		t.Error("SetVerbose(true) did not enable verbose mode")
	}

	// Test disabling verbose mode
	SetVerbose(false)
	if verboseEnabled {
		t.Error("SetVerbose(false) did not disable verbose mode")
	}
}

func TestFormattedMessages(t *testing.T) {
	// Test that formatted messages don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Formatted messages panicked: %v", r)
		}
	}()

	Info("file: %s, line: %d", "test.go", 42)
	Success("processed %d files", 10)
	Warning("skipped %s", "file.txt")
	Error("failed at line %d", 100)
}
