package opencode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			cw := &RingWriter{n: 10, lines: tt.lines, buf: tt.buf}
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
	assert.Contains(t, stdout.String(), "run output: run --format json --model test-model test-prompt")
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

func TestSessionCollector(t *testing.T) {
	c := &SessionCollector{}

	ids := c.IDs()
	assert.Empty(t, ids)

	c.Append("session-1")
	c.Append("session-2")

	ids = c.IDs()
	assert.Equal(t, []string{"session-1", "session-2"}, ids)
}

func TestSessionCollectorConcurrency(t *testing.T) {
	c := &SessionCollector{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Append(fmt.Sprintf("session-%d", n))
		}(i)
	}
	wg.Wait()
	assert.Len(t, c.IDs(), 100)
}

func TestSessionCollectorIDsIsCopy(t *testing.T) {
	c := &SessionCollector{}
	c.Append("session-1")

	ids := c.IDs()
	ids[0] = "tampered"

	assert.Equal(t, []string{"session-1"}, c.IDs())
}

func TestWithSessionCollector(t *testing.T) {
	ctx := context.Background()
	c := &SessionCollector{}

	ctx = WithSessionCollector(ctx, c)
	got := SessionCollectorFrom(ctx)
	assert.Same(t, c, got)
}

func TestSessionCollectorFromNil(t *testing.T) {
	ctx := context.Background()
	got := SessionCollectorFrom(ctx)
	assert.Nil(t, got)
}

func TestSessionParserExtractsID(t *testing.T) {
	var buf bytes.Buffer
	sp := newSessionParser(&buf)
	line := `{"sessionID":"sess-abc123","other":"data"}` + "\n"
	n, err := sp.Write([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, len(line), n)
	assert.Equal(t, "sess-abc123", sp.sessionID)
	assert.Contains(t, buf.String(), "sess-abc123")
}

func TestSessionParserExtractsFirstIDOnly(t *testing.T) {
	var buf bytes.Buffer
	sp := newSessionParser(&buf)
	sp.Write([]byte(`{"sessionID":"sess-first","x":1}` + "\n"))
	sp.Write([]byte(`{"sessionID":"sess-second","x":2}` + "\n"))
	assert.Equal(t, "sess-first", sp.sessionID)
}

func TestSessionParserSkipsNonJSON(t *testing.T) {
	var buf bytes.Buffer
	sp := newSessionParser(&buf)
	sp.Write([]byte("not json at all\n"))
	assert.Empty(t, sp.sessionID)
}

func TestSessionParserSkipsJSONWithoutSessionID(t *testing.T) {
	var buf bytes.Buffer
	sp := newSessionParser(&buf)
	sp.Write([]byte(`{"other":"data"}` + "\n"))
	assert.Empty(t, sp.sessionID)
}

func TestSessionParserSuppressesOutputWhenNilWriter(t *testing.T) {
	sp := newSessionParser(nil)
	line := `{"sessionID":"sess-abc123","other":"data"}` + "\n"
	sp.Write([]byte(line))
	assert.Equal(t, "sess-abc123", sp.sessionID)
}

func TestRunCommandCapturesSessionID(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	scriptContent := `#!/bin/bash
echo '{"sessionID":"sess-from-run","other":"data"}'
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	collector := &SessionCollector{}
	ctx := WithSessionCollector(context.Background(), collector)

	var stdout, stderr bytes.Buffer
	err = RunCommand(ctx, "test-model", "test-prompt", &stdout, &stderr)
	require.NoError(t, err)

	ids := collector.IDs()
	require.Len(t, ids, 1)
	assert.Equal(t, "sess-from-run", ids[0])
}

func TestRunCommandSuppressesOutputWhenNilStdoutWriter(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	scriptContent := `#!/bin/bash
echo '{"sessionID":"sess-suppressed","other":"data"}'
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	collector := &SessionCollector{}
	ctx := WithSessionCollector(context.Background(), collector)

	err = RunCommand(ctx, "test-model", "test-prompt", nil, nil)
	require.NoError(t, err)

	ids := collector.IDs()
	require.Len(t, ids, 1)
	assert.Equal(t, "sess-suppressed", ids[0])
}

func TestRunCommandNoSessionCollector(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	scriptContent := `#!/bin/bash
echo '{"sessionID":"sess-orphan","other":"data"}'
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
}
