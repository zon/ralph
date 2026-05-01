package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall(t *testing.T) {
	t.Run("creates directory and writes skill file", func(t *testing.T) {
		tmpDir := t.TempDir()

		skills := map[string]string{
			"ralph-write-spec": "# SKILL.md content for ralph-write-spec",
			"ralph-write-flow": "# SKILL.md content for ralph-write-flow",
		}

		err := Install(tmpDir, skills)
		require.NoError(t, err)

		for name, content := range skills {
			skillPath := filepath.Join(tmpDir, ".claude", "skills", name, "SKILL.md")
			data, err := os.ReadFile(skillPath)
			require.NoError(t, err)
			assert.Equal(t, content, string(data))
		}
	})

	t.Run("overwrites existing skill", func(t *testing.T) {
		tmpDir := t.TempDir()

		existingDir := filepath.Join(tmpDir, ".claude", "skills", "ralph-write-spec")
		err := os.MkdirAll(existingDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(existingDir, "SKILL.md"), []byte("old content"), 0644)
		require.NoError(t, err)

		skills := map[string]string{
			"ralph-write-spec": "new content",
		}

		err = Install(tmpDir, skills)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(existingDir, "SKILL.md"))
		require.NoError(t, err)
		assert.Equal(t, "new content", string(data))
	})

	t.Run("returns error on write failure", func(t *testing.T) {
		tmpDir := t.TempDir()

		skills := map[string]string{
			"ralph-write-spec": "content",
		}

		readOnlyDir := filepath.Join(tmpDir, ".claude", "skills", "ralph-write-spec")
		err := os.MkdirAll(readOnlyDir, 0755)
		require.NoError(t, err)

		err = Install("/proc/0/nonexistent", skills)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInstallFailed)
	})
}