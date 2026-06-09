package opencode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestIsFatalError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		fatal bool
	}{
		{name: "nil error", err: nil, fatal: false},
		{name: "Insufficient Balance", err: fmt.Errorf("Insufficient Balance"), fatal: true},
		{name: "insufficient balance", err: fmt.Errorf("insufficient balance"), fatal: true},
		{name: "billing", err: fmt.Errorf("billing error"), fatal: true},
		{name: "account", err: fmt.Errorf("account problem"), fatal: true},
		{name: "payment required", err: fmt.Errorf("payment required"), fatal: true},
		{name: "quota exceeded", err: fmt.Errorf("quota exceeded"), fatal: true},
		{name: "no match", err: fmt.Errorf("some random error"), fatal: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.fatal, IsFatalError(tt.err))
		})
	}
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

func TestRunAgent(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	outputPath := filepath.Join(tmpDir, "output.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
echo "agent ran successfully"
echo "done" > '%s'
exit 0
`, outputPath)
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	client := New()
	err = client.RunAgent(context.Background(), "test-model", "", "test-prompt")
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "done\n", string(data))
}

func TestRunAgentFailure(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
for i in $(seq 1 15); do
  echo "line $i"
done
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
	err = client.RunAgent(context.Background(), "test-model", "", "test-prompt")
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "opencode execution failed")
	assert.Contains(t, errMsg, "Last 10 lines of output:")

	prefix := "Last 10 lines of output:\n"
	idx := strings.Index(errMsg, prefix)
	require.NotEqual(t, -1, idx)
	tail := errMsg[idx+len(prefix):]

	var expectedLines []string
	for i := 6; i <= 15; i++ {
		expectedLines = append(expectedLines, fmt.Sprintf("line %d", i))
	}
	expectedTail := strings.Join(expectedLines, "\n")
	assert.Equal(t, expectedTail, tail)
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
if [ "$1" = "stats" ]; then
  echo ' Session Stats'
  echo ' Input           3.5M │'
  echo ' Output          542.0K │'
  echo ' Total Cost      $12.34 │'
  exit 0
fi
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
	stats, err := client.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(3500000), stats.InputTokens)
	assert.Equal(t, int64(542000), stats.OutputTokens)
	assert.InDelta(t, 12.34, stats.Cost, 0.001)
}

func TestDisplayStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

		scriptContent := `#!/bin/bash
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
	})

	t.Run("failure", func(t *testing.T) {
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

		client := New()
		err = client.DisplayStats()
		require.Error(t, err)
	})
}

func TestRunCommand_RealOpenCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	client := New()
	err := client.RunCommand(context.Background(), "deepseek/deepseek-v4-flash", "", "say hi", &stdout, &stderr)
	require.NoError(t, err)
}
