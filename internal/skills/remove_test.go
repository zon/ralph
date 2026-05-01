package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveStale(t *testing.T) {
	t.Run("removes stale ralph-prefixed skills", func(t *testing.T) {
		tmpDir := t.TempDir()

		skillsDir := filepath.Join(tmpDir, ".claude", "skills")
		err := os.MkdirAll(skillsDir, 0755)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(skillsDir, "ralph-old-skill"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(skillsDir, "ralph-old-skill", "SKILL.md"), []byte("old"), 0644)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(skillsDir, "ralph-write-spec"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(skillsDir, "ralph-write-spec", "SKILL.md"), []byte("spec content"), 0644)
		require.NoError(t, err)

		err = RemoveStale(tmpDir, []string{"ralph-write-spec"})
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(skillsDir, "ralph-old-skill"))
		assert.True(t, os.IsNotExist(err), "stale skill should be removed")

		_, err = os.Stat(filepath.Join(skillsDir, "ralph-write-spec"))
		assert.NoError(t, err, "current skill should remain")
	})

	t.Run("leaves non-ralph skills untouched", func(t *testing.T) {
		tmpDir := t.TempDir()

		skillsDir := filepath.Join(tmpDir, ".claude", "skills")
		err := os.MkdirAll(skillsDir, 0755)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(skillsDir, "my-custom-skill"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(skillsDir, "my-custom-skill", "SKILL.md"), []byte("custom"), 0644)
		require.NoError(t, err)

		err = RemoveStale(tmpDir, []string{"ralph-write-spec"})
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(skillsDir, "my-custom-skill"))
		assert.NoError(t, err, "non-ralph skill should remain")
	})

	t.Run("handles missing skills directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := RemoveStale(tmpDir, []string{"ralph-write-spec"})
		require.NoError(t, err)
	})

	t.Run("handles empty skills directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		skillsDir := filepath.Join(tmpDir, ".claude", "skills")
		err := os.MkdirAll(skillsDir, 0755)
		require.NoError(t, err)

		err = RemoveStale(tmpDir, []string{"ralph-write-spec"})
		require.NoError(t, err)
	})
}