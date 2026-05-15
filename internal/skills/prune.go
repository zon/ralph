package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func PruneStale(root string, fetched []Skill) {
	skillsDir := filepath.Join(root, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return
	}

	fetchedNames := make(map[string]bool)
	for _, skill := range fetched {
		fetchedNames[skill.Name] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "ralph-") {
			continue
		}
		if fetchedNames[name] {
			continue
		}
		skillDir := filepath.Join(skillsDir, name)
		if err := os.RemoveAll(skillDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove stale skill %s: %v\n", name, err)
		}
	}
}