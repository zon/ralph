package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// deleteFile removes a file from the filesystem and stages the deletion
func deleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file '%s': %w", filePath, err)
	}

	_, err := runGit("rm", filePath)
	if err != nil {
		return fmt.Errorf("failed to stage deletion of '%s': %w", filePath, err)
	}

	return nil
}

// DetectModifiedProjectFile finds the first modified or new YAML file in the projects directory.
// Returns the absolute path to the modified project file, or empty string if none found.
func DetectModifiedProjectFile(projectsDir string) (string, error) {
	files, err := DetectAllModifiedProjectFiles(projectsDir)
	if err != nil {
		return "", err
	}
	if len(files) > 0 {
		return files[0], nil
	}
	return "", nil
}

// DetectAllModifiedProjectFiles finds all modified or new YAML files in the projects directory.
// Returns a list of absolute paths to modified project files, or an empty list if none found.
func DetectAllModifiedProjectFiles(projectsDir string) ([]string, error) {
	absProjectsDir, err := filepath.Abs(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve projects directory: %w", err)
	}

	entries, err := os.ReadDir(absProjectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var modifiedFiles []string
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
			modifiedFiles = append(modifiedFiles, filePath)
		}
	}

	return modifiedFiles, nil
}
