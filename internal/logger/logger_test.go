package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	require.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.level)
	assert.False(t, l.json)
	assert.False(t, l.verbose)
}

func TestNewLoggerWithOptions(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	require.NotNil(t, l)
	assert.True(t, l.json)
	assert.Equal(t, buf, l.out)
}

func TestWithJSONOutput(t *testing.T) {
	l := NewLogger(WithJSONOutput())
	assert.True(t, l.json)
}

func TestWithOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	assert.Equal(t, buf, l.out)
}

func TestLoggerSetVerbose(t *testing.T) {
	l := NewLogger()
	assert.False(t, l.verbose)
	l.SetVerbose(true)
	assert.True(t, l.verbose)
	l.SetVerbose(false)
	assert.False(t, l.verbose)
}

func TestLoggerLevel(t *testing.T) {
	l := NewLogger()
	assert.Equal(t, LevelInfo, l.level)
}

func TestLevelString(t *testing.T) {
	assert.Equal(t, "DEBUG", LevelDebug.String())
	assert.Equal(t, "INFO", LevelInfo.String())
	assert.Equal(t, "WARN", LevelWarn.String())
	assert.Equal(t, "ERROR", LevelError.String())
	assert.Equal(t, "UNKNOWN", Level(100).String())
}

func TestLoggerInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Info("test message")
	assert.Contains(t, buf.String(), "[INFO] test message")
}

func TestLoggerInfof(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Infof("formatted: %s, %d", "test", 42)
	assert.Contains(t, buf.String(), "[INFO] formatted: test, 42")
}

func TestLoggerDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.level = LevelDebug
	l.Debug("debug message")
	assert.Contains(t, buf.String(), "[DEBUG] debug message")
}

func TestLoggerDebugf(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.level = LevelDebug
	l.Debugf("debug: %s", "test")
	assert.Contains(t, buf.String(), "[DEBUG] debug: test")
}

func TestLoggerWarn(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Warn("warning message")
	assert.Contains(t, buf.String(), "[WARN] warning message")
}

func TestLoggerWarnf(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Warnf("warning: %s", "test")
	assert.Contains(t, buf.String(), "[WARN] warning: test")
}

func TestLoggerError(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Error("error message")
	assert.Contains(t, buf.String(), "[ERROR] error message")
}

func TestLoggerErrorf(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Errorf("error: %s", "test")
	assert.Contains(t, buf.String(), "[ERROR] error: test")
}

func TestLoggerSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Success("success message")
	assert.Contains(t, buf.String(), "[SUCCESS] success message")
}

func TestLoggerSuccessf(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Successf("success: %d", 42)
	assert.Contains(t, buf.String(), "[SUCCESS] success: 42")
}

func TestLoggerVerboseWhenDisabled(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.SetVerbose(false)
	l.Verbose("verbose message")
	assert.Empty(t, buf.String())
}

func TestLoggerVerboseWhenEnabled(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.SetVerbose(true)
	l.Verbose("verbose message")
	assert.Contains(t, buf.String(), "[VERBOSE] verbose message")
}

func TestLoggerVerbosefWhenEnabled(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.SetVerbose(true)
	l.Verbosef("verbose: %s", "test")
	assert.Contains(t, buf.String(), "[VERBOSE] verbose: test")
}

func TestLoggerVerbosefWhenDisabled(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.SetVerbose(false)
	l.Verbosef("verbose: %s", "test")
	assert.Empty(t, buf.String())
}

func TestLoggerJSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.Info("json message")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "json message", entry["msg"])
}

func TestLoggerJSONDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.level = LevelDebug
	l.Debug("debug json")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "DEBUG", entry["level"])
	assert.Equal(t, "debug json", entry["msg"])
}

func TestLoggerJSONWarn(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.Warn("warn json")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "WARN", entry["level"])
	assert.Equal(t, "warn json", entry["msg"])
}

func TestLoggerJSONError(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.Error("error json")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "ERROR", entry["level"])
	assert.Equal(t, "error json", entry["msg"])
}

func TestLoggerJSONSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.Success("success json")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", entry["level"])
	assert.Equal(t, "success json", entry["msg"])
}

func TestLoggerJSONVerbose(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf), WithJSONOutput())
	l.SetVerbose(true)
	l.Verbose("verbose json")
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err)
	assert.Equal(t, "VERBOSE", entry["level"])
	assert.Equal(t, "verbose json", entry["msg"])
}

func TestLoggerLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.level = LevelWarn

	l.Debug("should not appear")
	assert.Empty(t, buf.String())

	l.Info("should not appear")
	assert.Empty(t, buf.String())

	l.Warn("should appear")
	assert.Contains(t, buf.String(), "[WARN] should appear")

	buf.Reset()
	l.Error("should appear")
	assert.Contains(t, buf.String(), "[ERROR] should appear")
}

func TestLoggerInfofWhenFiltered(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.level = LevelError

	l.Infof("should not appear: %s", "test")
	assert.Empty(t, buf.String())
}

func TestLoggerWithOutputWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.Info("direct output")
	assert.Contains(t, buf.String(), "direct output")
}

func TestLoggerSetVerboseOnNewLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger(WithOutput(buf))
	l.SetVerbose(true)
	l.Verbose("verbose on new logger")
	assert.Contains(t, buf.String(), "[VERBOSE] verbose on new logger")
}
