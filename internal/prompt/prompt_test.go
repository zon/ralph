package prompt

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderKey_ReturnsKey(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	go func() {
		stdinW.WriteString("sk-ant-123\n")
		stdinW.Close()
	}()

	key, err := ProviderKey("anthropic")
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-123", key)

	stdoutW.Close()
	io.Copy(io.Discard, stdoutR)
}

func TestProviderKey_PrintsPrompt(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	go func() {
		stdinW.WriteString("sk-ant-123\n")
		stdinW.Close()
	}()

	_, err = ProviderKey("anthropic")
	require.NoError(t, err)

	stdoutW.Close()
	promptOutput, err := io.ReadAll(stdoutR)
	require.NoError(t, err)
	assert.Equal(t, "Enter API key for anthropic: ", string(promptOutput))
}

func TestProviderKey_BlankString_ReturnsError(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	go func() {
		stdinW.WriteString("\n")
		stdinW.Close()
	}()

	_, err = ProviderKey("anthropic")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot be blank"))

	stdoutW.Close()
	io.Copy(io.Discard, stdoutR)
}

func TestProviderKey_WhitespaceOnly_ReturnsError(t *testing.T) {
	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	go func() {
		stdinW.WriteString("   \t  \n")
		stdinW.Close()
	}()

	_, err = ProviderKey("deepseek")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot be blank"))

	stdoutW.Close()
	io.Copy(io.Discard, stdoutR)
}
