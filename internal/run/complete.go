package run

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/file"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

// FindCompleteProjects scans a directory for project YAML files where all requirements have passing true.
// A project file with zero requirements is not considered complete.
// Returns a slice of absolute file paths to complete project files.
func FindCompleteProjects(dir string) ([]string, error) {
	var completeProjects []string

	// Check if directory exists
	if _, err := file.Stat(dir); file.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	// Find all YAML files recursively in the directory tree
	var allFiles []string
	err := file.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := file.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Check each file
	for _, filePath := range allFiles {
		project, err := project.LoadProject(filePath)
		if err != nil {
			// Skip files that can't be loaded as valid projects
			continue
		}

		// Check if project is complete
		if isProjectComplete(project) {
			absPath, err := file.Abs(filePath)
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
func isProjectComplete(project *project.Project) bool {
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

	// Delete each file
	for _, filePath := range files {
		if err := file.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove project file %s: %w", filePath, err)
		}
		logger.Infof("Removed complete project file: %s", filePath)
	}

	// Stage the deletions
	for _, filePath := range files {
		if err := git.StageFile(filePath); err != nil {
			return fmt.Errorf("failed to stage deleted file %s: %w", filePath, err)
		}
	}

	// Create commit
	commitMessage := "chore: remove complete project files"
	if err := git.Commit(commitMessage); err != nil {
		return fmt.Errorf("failed to commit deleted files: %w", err)
	}

	logger.Successf("Committed removal of %d complete project file(s)", len(files))
	return nil
}
