package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

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
