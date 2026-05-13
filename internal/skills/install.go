package skills

import (
	"os"
	"path/filepath"
)

func InstallAll(root string, fetched []Skill) error {
	skillsDir := filepath.Join(root, ".claude", "skills")

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	for _, skill := range fetched {
		skillDir := filepath.Join(skillsDir, skill.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(skill.Body), 0644); err != nil {
			return err
		}
	}

	return nil
}