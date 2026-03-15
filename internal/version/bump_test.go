package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBumpPatchIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	versionFile := filepath.Join(tmpDir, "VERSION")

	originalVersionFile := "internal/version/VERSION"
	versionFilePath = originalVersionFile
	defer func() {
		versionFilePath = originalVersionFile
	}()

	versionFilePath = versionFile

	err := os.WriteFile(versionFile, []byte("5.1.2\n"), 0644)
	require.NoError(t, err)

	err = BumpPatch()
	require.NoError(t, err)

	versionData, err := os.ReadFile(versionFile)
	require.NoError(t, err)
	require.Equal(t, "5.1.3\n", string(versionData))
}
