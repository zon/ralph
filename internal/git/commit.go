package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// StageFile stages a specific file using git add
func StageFile(filePath string) error {
	cmd := exec.Command("git", "add", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file '%s': %w (output: %s)", filePath, err, out.String())
	}

	return nil
}

// StageAll stages all changes using git add -A
func StageAll() error {
	cmd := exec.Command("git", "add", "-A")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes: %w (output: %s)", err, out.String())
	}

	return nil
}

// HasFileChanges checks if a specific file has unstaged changes (i.e. differs from the index)
func HasFileChanges(filePath string) bool {
	// git diff --quiet -- <file>: exit 0 = no changes, exit 1 = has changes
	cmd := exec.Command("git", "diff", "--quiet", "--", filePath)
	err := cmd.Run()
	return err != nil
}

// HasStagedChanges checks if there are any staged changes ready to commit
func HasStagedChanges() bool {
	// Use git diff --cached --quiet to check for staged changes
	// Exit code 0 = no staged changes, exit code 1 = has staged changes
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	return err != nil // Non-zero exit = has changes
}

// HasUncommittedChanges reports whether there are any uncommitted changes in the
// working tree or index (i.e. staged or unstaged modifications, additions, or deletions).
func HasUncommittedChanges() bool {
	// git status --porcelain: empty output = clean, non-empty = dirty
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(out.String()) != ""
}

// GetDiffSince returns the diff between the base branch and HEAD
func GetDiffSince(base string) (string, error) {
	// Use git diff to get changes between base and HEAD
	cmd := exec.Command("git", "diff", fmt.Sprintf("%s..HEAD", base))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get diff since '%s': %w (output: %s)", base, err, out.String())
	}

	diff := out.String()

	return diff, nil
}

// Commit creates a git commit with the specified message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w (output: %s)", err, out.String())
	}

	return nil
}

// CommitAllowEmpty creates a git commit even when there are no staged changes.
// Used when the AI ran but produced no file changes (e.g. all requirements already passing).
func CommitAllowEmpty(message string) error {
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", message)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit (allow-empty): %w (output: %s)", err, out.String())
	}

	return nil
}

// CommitChanges stages all changes and commits them with a descriptive message
// It generates the commit message based on changed files
// Returns error if there are no changes to commit
func CommitChanges() error {
	// Stage all changes
	if err := StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are any staged changes
	if !HasStagedChanges() {
		return fmt.Errorf("no changes to commit")
	}

	// Get list of changed files to generate commit message
	message, err := generateCommitMessage()
	if err != nil {
		// Fallback to generic message if generation fails
		message = "Update project files"
	}

	// Commit the staged changes
	if err := Commit(message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// generateCommitMessage creates a descriptive commit message from changed files
func generateCommitMessage() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get staged files: %w (output: %s)", err, out.String())
	}

	stagedFiles := strings.TrimSpace(out.String())
	if stagedFiles == "" {
		return "Update files", nil
	}

	files := strings.Split(stagedFiles, "\n")
	fileCount := len(files)

	return buildCommitMessage(files, fileCount), nil
}

func buildCommitMessage(files []string, fileCount int) string {
	switch {
	case fileCount == 1:
		return fmt.Sprintf("Update %s", files[0])
	case fileCount <= 3:
		return fmt.Sprintf("Update %s", strings.Join(files, ", "))
	default:
		return summarizeCommitMessage(files, fileCount)
	}
}

func summarizeCommitMessage(files []string, fileCount int) string {
	categories := categorizeFiles(files)
	if len(categories) == 1 {
		for category := range categories {
			return fmt.Sprintf("Update %s files (%d files)", category, fileCount)
		}
	}
	return fmt.Sprintf("Update %d files across project", fileCount)
}

// categorizeFile categorizes a single file by directory or extension
func categorizeFile(file string) string {
	if strings.Contains(file, "/") {
		parts := strings.Split(file, "/")
		if len(parts) > 1 {
			return parts[0]
		}
	}
	if strings.Contains(file, ".") {
		parts := strings.Split(file, ".")
		return parts[len(parts)-1]
	}
	return "root"
}

// categorizeFiles groups files by directory or type
func categorizeFiles(files []string) map[string]int {
	categories := make(map[string]int)

	for _, file := range files {
		category := categorizeFile(file)
		categories[category]++
	}

	return categories
}

// GetCommitLog retrieves commit log formatted exactly like the reference implementation.
// Returns a single string with commits formatted as "%h: %B" (hash: full message).
// Gets all commits since base..HEAD. If limit > 0, only the most recent limit commits are returned.
func GetCommitLog(base string, limit int) (string, error) {
	// Use git log with format matching reference: %h: %B (hash: full message body)
	logRange := fmt.Sprintf("%s..HEAD", base)
	args := []string{"log", logRange, "--format=%h: %B"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit log: %w (output: %s)", err, out.String())
	}

	output := strings.TrimSpace(out.String())
	return output, nil
}
