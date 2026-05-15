package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPruneStale(t *testing.T) {
	t.Run("removes ralph-prefixed skill not in fetched set", func(t *testing.T) {
		root := t.TempDir()
		skillDir := filepath.Join(root, ".claude", "skills", "ralph-old-skill")
		err := os.MkdirAll(skillDir, 0755)
		require.NoError(t, err)
		skillPath := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillPath, []byte("old content"), 0644)
		require.NoError(t, err)

		fetched := []Skill{
			{Name: "ralph-write-spec", Content: "# Spec\nContent"},
		}

		PruneStale(root, fetched)

		_, err = os.Stat(skillDir)
		require.True(t, os.IsNotExist(err), "ralph-old-skill should have been removed")
	})

	t.Run("leaves non-ralph skills untouched", func(t *testing.T) {
		root := t.TempDir()
		skillDir := filepath.Join(root, ".claude", "skills", "my-custom-skill")
		err := os.MkdirAll(skillDir, 0755)
		require.NoError(t, err)
		skillPath := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillPath, []byte("custom content"), 0644)
		require.NoError(t, err)

		fetched := []Skill{
			{Name: "ralph-write-spec", Content: "# Spec\nContent"},
		}

		PruneStale(root, fetched)

		_, err = os.Stat(skillPath)
		require.NoError(t, err, "my-custom-skill should remain untouched")
	})
}