package eino

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodingTools(t *testing.T) {
	tools := CodingTools()
	require.Len(t, tools, 3)

	names := make([]string, len(tools))
	for i, tool := range tools {
		info, err := tool.Info(context.Background())
		require.NoError(t, err)
		names[i] = info.Name
	}
	assert.ElementsMatch(t, []string{"read", "write", "bash"}, names)
}

func TestReadFileTool(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	path := filepath.Join(dir, "test.txt")
	err := os.WriteFile(path, []byte("hello world"), 0644)
	require.NoError(t, err)

	t.Run("read existing file", func(t *testing.T) {
		result, err := readFile(ctx, path)
		require.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := readFile(ctx, filepath.Join(dir, "nope.txt"))
		require.Error(t, err)
	})
}

func TestWriteFileTool(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	t.Run("write creates file with content", func(t *testing.T) {
		path := filepath.Join(dir, "out.txt")
		result, err := writeFile(ctx, path, "some content")
		require.NoError(t, err)
		assert.Equal(t, "ok", result)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "some content", string(data))
	})

	t.Run("write to non-existent directory returns error", func(t *testing.T) {
		_, err := writeFile(ctx, filepath.Join(dir, "nope", "file.txt"), "x")
		require.Error(t, err)
	})
}

func TestBashTool(t *testing.T) {
	ctx := context.Background()

	t.Run("echo command", func(t *testing.T) {
		result, err := runBash(ctx, "echo hello")
		require.NoError(t, err)
		assert.Contains(t, result, "hello")
	})

	t.Run("command with stderr output", func(t *testing.T) {
		result, err := runBash(ctx, "echo out && echo err >&2")
		require.NoError(t, err)
		assert.Contains(t, result, "out")
		assert.Contains(t, result, "err")
	})

	t.Run("failing command returns error", func(t *testing.T) {
		_, err := runBash(ctx, "exit 42")
		require.Error(t, err)
	})
}

func readFile(ctx context.Context, path string) (string, error) {
	tool, err := newReadFileTool()
	if err != nil {
		return "", err
	}
	return tool.InvokableRun(ctx, `{"path":"`+path+`"}`)
}

func writeFile(ctx context.Context, path, content string) (string, error) {
	tool, err := newWriteFileTool()
	if err != nil {
		return "", err
	}
	return tool.InvokableRun(ctx, `{"path":"`+path+`","content":"`+content+`"}`)
}

func runBash(ctx context.Context, command string) (string, error) {
	tool, err := newBashTool()
	if err != nil {
		return "", err
	}
	return tool.InvokableRun(ctx, `{"command":"`+command+`"}`)
}
