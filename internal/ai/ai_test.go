package ai

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

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
			result := cw.tail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunAgentErrorIncludesTail(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "line 1 output"
echo "line 2 output"
echo "line 3 output"
echo "line 4 output"
echo "line 5 output"
echo "line 6 output"
echo "line 7 output"
echo "line 8 output"
echo "line 9 output"
echo "line 10 output"
echo "line 11 output"
echo "line 12 output"
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)
	t.Setenv("RALPH_MOCK_AI", "")

	ctx := testutil.NewContext()
	err = RunAgent(ctx, "test prompt")

	require.Error(t, err, "RunAgent should return error when opencode fails")
	assert.Contains(t, err.Error(), "opencode execution failed")
	assert.Contains(t, err.Error(), "line 3")
	assert.Contains(t, err.Error(), "line 12")
	assert.NotContains(t, err.Error(), "line 2 output", "Should not include lines before last 10")
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

	err = DisplayStats()
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
	err = RunCommand(context.Background(), "test-model", "test-prompt", &stdout, &stderr)
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
	err = RunCommand(context.Background(), "test-model", "test-prompt", &stdout, &stderr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode command failed")
}
