package logger

import (
	"testing"
)

// Note: Testing logger output directly is challenging because the color package
// writes to the underlying file descriptors. Instead, we test the behavior
// and state management.

func TestInfo(t *testing.T) {
	// Test that Info doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Info panicked: %v", r)
		}
	}()
	Info("test message")
	Infof("formatted: %s, %d", "test", 42)
}

func TestSuccess(t *testing.T) {
	// Test that Success doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Success panicked: %v", r)
		}
	}()
	Success("operation completed")
	Successf("completed: %d items", 5)
}

func TestWarning(t *testing.T) {
	// Test that Warning doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Warning panicked: %v", r)
		}
	}()
	Warning("potential issue")
	Warningf("warning: %s", "something")
}

func TestError(t *testing.T) {
	// Test that Error doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Error panicked: %v", r)
		}
	}()
	Error("something failed")
	Errorf("error: %v", "details")
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
	Verbosef("debug: %s", "test")
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
	Verbosef("debug: %d", 123)
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

	Infof("file: %s, line: %d", "test.go", 42)
	Successf("processed %d files", 10)
	Warningf("skipped %s", "file.txt")
	Errorf("failed at line %d", 100)
}
