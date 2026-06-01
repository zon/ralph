package opencode

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOCClientInterface(t *testing.T) {
	var _ OCClient = (*Client)(nil)
	var _ OCClient = (*MockOC)(nil)
}

func TestNewClient(t *testing.T) {
	c := New()
	require.NotNil(t, c)
	var _ OCClient = c
}

func TestStatsParsing(t *testing.T) {
	output := ` Session Stats
 Input           3.5M │
 Output          542.0K │
 Total Cost      $12.34 │
`
	stats, err := parseStatsOutput(output)
	require.NoError(t, err)
	assert.Equal(t, int64(3500000), stats.InputTokens)
	assert.Equal(t, int64(542000), stats.OutputTokens)
	assert.InDelta(t, 12.34, stats.Cost, 0.001)
}

func TestStatsParsingZeroValues(t *testing.T) {
	output := ` Session Stats
 Input           0 │
 Output          0 │
 Total Cost      $0.00 │
`
	stats, err := parseStatsOutput(output)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.InputTokens)
	assert.Equal(t, int64(0), stats.OutputTokens)
	assert.InDelta(t, 0.00, stats.Cost, 0.001)
}

func TestStatsParsingKMValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1K", 1000},
		{"1.5K", 1500},
		{"542.0K", 542000},
		{"1M", 1000000},
		{"3.5M", 3500000},
		{"10.123M", 10123000},
		{"0K", 0},
		{"0M", 0},
		{"123", 123},
		{"0.5K", 500},
	}

	for _, tt := range tests {
		val, err := parseTokenValue(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, val, "input: %s", tt.input)
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo ' Input           3.5M │'
echo ' Output          542.0K │'
echo ' Total Cost      $12.34 │'
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	client := New()
	stats, err := client.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(3500000), stats.InputTokens)
	assert.Equal(t, int64(542000), stats.OutputTokens)
	assert.InDelta(t, 12.34, stats.Cost, 0.001)
}

func TestGetStatsError(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "err output" >&2
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	client := New()
	_, err = client.GetStats()
	require.Error(t, err)
}

func TestGetStatsParseError(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo 'no stats here'
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	client := New()
	_, err = client.GetStats()
	require.Error(t, err)
}

func TestCaptureWriterTail(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		buf      string
		expected string
	}{
		{
			name:     "empty",
			lines:    []string{},
			buf:      "",
			expected: "",
		},
		{
			name:     "fewer than n lines",
			lines:    []string{"line1", "line2", "line3"},
			buf:      "",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "with partial line",
			lines:    []string{"line1", "line2"},
			buf:      "line3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "exactly n lines",
			lines:    []string{"line1", "line2", "line3"},
			buf:      "",
			expected: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cw := &ringWriter{n: 10, lines: tt.lines, buf: tt.buf}
			result := cw.Tail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDisplayStats(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "stats output line 1"
echo "stats output line 2"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	client := New()
	err = client.DisplayStats()
	require.NoError(t, err)
}

func TestRunCommand(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "run output: $@"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	var stdout, stderr bytes.Buffer
	client := New()
	err = client.RunCommand(context.Background(), "test-model", "", "test-prompt", &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "run output: run --model test-model test-prompt")
}

func TestRunCommandFailure(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "error output" >&2
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	var stdout, stderr bytes.Buffer
	client := New()
	err = client.RunCommand(context.Background(), "test-model", "", "test-prompt", &stdout, &stderr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode command failed")
}
