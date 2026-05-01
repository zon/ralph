package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RemoveStale(repoRoot string, currentSkills []string) error {
	skillsDir := filepath.Join(repoRoot, ".claude", "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	currentSet := make(map[string]bool)
	for _, s := range currentSkills {
		currentSet[s] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "ralph-") {
			continue
		}

		if !currentSet[name] {
			skillPath := filepath.Join(skillsDir, name)
			if err := os.RemoveAll(skillPath); err != nil {
				return fmt.Errorf("failed to remove stale skill %s: %w", name, err)
			}
		}
	}

	return nil
}