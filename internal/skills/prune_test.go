package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPruneStale_RemovesStaleRalphSkill(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-old-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "ralph-old-skill", "SKILL.md"), []byte("old"), 0644))

	PruneStale(root, []Skill{})

	_, err := os.Stat(filepath.Join(skillsDir, "ralph-old-skill"))
	require.True(t, os.IsNotExist(err), "ralph-old-skill should have been removed")
}

func TestPruneStale_LeavesNonRalphSkillsUntouched(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "my-custom-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "my-custom-skill", "SKILL.md"), []byte("mine"), 0644))

	PruneStale(root, []Skill{{Name: "ralph-write-spec", Body: "spec content"}})

	content, err := os.ReadFile(filepath.Join(skillsDir, "my-custom-skill", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "mine", string(content))
}

func TestPruneStale_PreservesFetchedRalphSkills(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-write-spec"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "ralph-write-spec", "SKILL.md"), []byte("old"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-old-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "ralph-old-skill", "SKILL.md"), []byte("stale"), 0644))

	PruneStale(root, []Skill{{Name: "ralph-write-spec", Body: "new"}})

	_, err := os.Stat(filepath.Join(skillsDir, "ralph-write-spec"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(skillsDir, "ralph-old-skill"))
	require.True(t, os.IsNotExist(err), "ralph-old-skill should be removed")
}