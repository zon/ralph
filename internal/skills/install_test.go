package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallAll(t *testing.T) {
	t.Run("creates skills directory and writes skill files", func(t *testing.T) {
		root := t.TempDir()
		fetched := []Skill{
			{Name: "ralph-write-spec", Content: "# Write Spec\nSome content"},
			{Name: "ralph-write-flow", Content: "# Write Flow\nOther content"},
		}

		err := InstallAll(root, fetched)
		require.NoError(t, err)

		for _, skill := range fetched {
			skillPath := filepath.Join(root, ".claude", "skills", skill.Name, "SKILL.md")
			content, err := os.ReadFile(skillPath)
			require.NoError(t, err, "skill %s should be installed", skill.Name)
			require.Equal(t, skill.Content, string(content))
		}
	})

	t.Run("overwrites existing skills", func(t *testing.T) {
		root := t.TempDir()
		existingPath := filepath.Join(root, ".claude", "skills", "ralph-write-spec", "SKILL.md")
		err := os.MkdirAll(filepath.Dir(existingPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(existingPath, []byte("old content"), 0644)
		require.NoError(t, err)

		fetched := []Skill{
			{Name: "ralph-write-spec", Content: "new content"},
		}

		err = InstallAll(root, fetched)
		require.NoError(t, err)

		content, err := os.ReadFile(existingPath)
		require.NoError(t, err)
		require.Equal(t, "new content", string(content))
	})
}