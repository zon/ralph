package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	assert.NotPanics(t, func() {
		Info("test message")
		Infof("formatted: %s, %d", "test", 42)
	})
}

func TestSuccess(t *testing.T) {
	assert.NotPanics(t, func() {
		Success("operation completed")
		Successf("completed: %d items", 5)
	})
}

func TestWarning(t *testing.T) {
	assert.NotPanics(t, func() {
		Warning("potential issue")
		Warningf("warning: %s", "something")
	})
}

func TestError(t *testing.T) {
	assert.NotPanics(t, func() {
		Error("something failed")
		Errorf("error: %v", "details")
	})
}

func TestVerboseWhenDisabled(t *testing.T) {
	SetVerbose(false)

	assert.NotPanics(t, func() {
		Verbose("debug info")
		Verbosef("debug: %s", "test")
	})
}

func TestVerboseWhenEnabled(t *testing.T) {
	SetVerbose(true)
	defer SetVerbose(false)

	assert.NotPanics(t, func() {
		Verbose("debug info")
		Verbosef("debug: %d", 123)
	})
}

func TestSetVerbose(t *testing.T) {
	SetVerbose(true)
	assert.True(t, verboseEnabled, "SetVerbose(true) should enable verbose mode")

	SetVerbose(false)
	assert.False(t, verboseEnabled, "SetVerbose(false) should disable verbose mode")
}

func TestFormattedMessages(t *testing.T) {
	assert.NotPanics(t, func() {
		Infof("file: %s, line: %d", "test.go", 42)
		Successf("processed %d files", 10)
		Warningf("skipped %s", "file.txt")
		Errorf("failed at line %d", 100)
	})
}
