package opencode

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	// This test requires the 'opencode' command to be available in the PATH.
	// For actual unit testing, consider mocking os/exec.Command or using a fake.

	// Test a successful command (assuming 'opencode' exists and 'opencode help' works)
	t.Run("successful command", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := RunCommand(context.Background(), "test-model", "test-prompt", &stdout, &stderr)
		// For a real test, you'd assert on stdout/stderr content as well.
		// As opencode run requires a real model and prompt, we'll mock the command for a true unit test
		// For now, we'll assume a successful execution means no error.
		require.NoError(t, err)
		assert.NotEmpty(t, stdout.String())
		assert.Empty(t, stderr.String())
	})

	// Test a command that might fail (e.g., non-existent model or invalid prompt)
	t.Run("failing command", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		// Assuming "nonexistent-model" will cause opencode to fail
		err := RunCommand(context.Background(), "nonexistent-model", "invalid-prompt", &stdout, &stderr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "opencode command failed")
	})
}
