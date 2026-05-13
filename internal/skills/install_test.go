package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallAll_CreatesSkillsDirectory(t *testing.T) {
	root := t.TempDir()

	err := InstallAll(root, []Skill{})

	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, ".claude", "skills"))
	require.NoError(t, err, ".claude/skills/ should be created")
}

func TestInstallAll_WritesSkillFiles(t *testing.T) {
	root := t.TempDir()

	fetched := []Skill{
		{Name: "ralph-write-spec", Body: "# Write Spec\nThis skill writes specs."},
		{Name: "ralph-write-flow", Body: "# Write Flow\nThis skill writes flows."},
	}

	err := InstallAll(root, fetched)

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(root, ".claude", "skills", "ralph-write-spec", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "# Write Spec\nThis skill writes specs.", string(content))
	content, err = os.ReadFile(filepath.Join(root, ".claude", "skills", "ralph-write-flow", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "# Write Flow\nThis skill writes flows.", string(content))
}

func TestInstallAll_OverwritesExistingSkills(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-write-spec"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "ralph-write-spec", "SKILL.md"), []byte("old"), 0644))

	err := InstallAll(root, []Skill{{Name: "ralph-write-spec", Body: "new"}})

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(skillsDir, "ralph-write-spec", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))
}

func TestInstallAll_LeavesNonTargetSkillsUntouched(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "my-custom-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "my-custom-skill", "SKILL.md"), []byte("mine"), 0644))

	err := InstallAll(root, []Skill{{Name: "ralph-write-spec", Body: "new"}})

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(skillsDir, "my-custom-skill", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "mine", string(content))
}