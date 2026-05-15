package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

func InstallAll(root string, fetched []Skill) error {
	for _, skill := range fetched {
		skillDir := filepath.Join(root, ".claude", "skills", skill.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory %s: %w", skillDir, err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(skill.Content), 0644); err != nil {
			return fmt.Errorf("failed to write skill file %s: %w", skillPath, err)
		}
	}
	return nil
}