package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// FindCompleteProjects scans a directory for project YAML files where all requirements have passing true.
// A project file with zero requirements is not considered complete.
// Returns a slice of absolute file paths to complete project files.
func FindCompleteProjects(dir string) ([]string, error) {
	var completeProjects []string

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	// Find all YAML files in the directory
	pattern := filepath.Join(dir, "*.yaml")
	yamlFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find YAML files: %w", err)
	}

	// Also find .yml files
	pattern = filepath.Join(dir, "*.yml")
	ymlFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find YML files: %w", err)
	}

	// Combine both lists
	allFiles := append(yamlFiles, ymlFiles...)

	// Check each file
	for _, file := range allFiles {
		project, err := config.LoadProject(file)
		if err != nil {
			// Skip files that can't be loaded as valid projects
			continue
		}

		// Check if project is complete
		if isProjectComplete(project) {
			absPath, err := filepath.Abs(file)
			if err != nil {
				// Skip files with path resolution errors
				continue
			}
			completeProjects = append(completeProjects, absPath)
		}
	}

	return completeProjects, nil
}

// isProjectComplete checks if a project has all requirements passing.
// A project with zero requirements is not considered complete.
func isProjectComplete(project *config.Project) bool {
	if len(project.Requirements) == 0 {
		return false
	}

	for _, req := range project.Requirements {
		if !req.Passing {
			return false
		}
	}

	return true
}

// RemoveAndCommit deletes the given project files, stages the deletions, and commits with message "chore: remove complete project files".
// If the files slice is empty the function is a no-op and returns nil.
func RemoveAndCommit(ctx *context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}

	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would remove complete project files and commit")
		for _, file := range files {
			logger.Infof("[DRY-RUN] Would remove: %s", file)
		}
		return nil
	}

	// Delete each file
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove project file %s: %w", file, err)
		}
		logger.Infof("Removed complete project file: %s", file)
	}

	// Stage the deletions
	for _, file := range files {
		if err := git.StageFile(ctx, file); err != nil {
			return fmt.Errorf("failed to stage deleted file %s: %w", file, err)
		}
	}

	// Create commit
	commitMessage := "chore: remove complete project files"
	if err := git.Commit(ctx, commitMessage); err != nil {
		return fmt.Errorf("failed to commit deleted files: %w", err)
	}

	logger.Successf("Committed removal of %d complete project file(s)", len(files))
	return nil
}
