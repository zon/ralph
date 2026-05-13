package skills

import (
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

	fetchedMap := make(map[string]bool)
	for _, s := range fetched {
		fetchedMap[s.Name] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "ralph-") {
			continue
		}
		if !fetchedMap[entry.Name()] {
			os.RemoveAll(filepath.Join(skillsDir, entry.Name()))
		}
	}
}