package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectModifiedProjectFile finds the first modified or new YAML file in the projects directory.
// Returns the absolute path to the modified project file, or empty string if none found.
func DetectModifiedProjectFile(projectsDir string) (string, error) {
	absProjectsDir, err := filepath.Abs(projectsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve projects directory: %w", err)
	}

	entries, err := os.ReadDir(absProjectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		filePath := filepath.Join(absProjectsDir, name)
		if IsFileModifiedOrNew(filePath) {
			return filePath, nil
		}
	}

	return "", nil
}
