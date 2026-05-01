package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

func Install(repoRoot string, skills map[string]string) error {
	skillsDir := filepath.Join(repoRoot, ".claude", "skills")

	for name, content := range skills {
		skillPath := filepath.Join(skillsDir, name, "SKILL.md")

		dir := filepath.Dir(skillPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("%w: failed to create directory %s: %v", ErrInstallFailed, dir, err)
		}

		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("%w: failed to write %s: %v", ErrInstallFailed, skillPath, err)
		}
	}

	return nil
}