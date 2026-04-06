package git

import (
	"fmt"
	"strings"
)

// StageFile stages a specific file using git add
func StageFile(filePath string) error {
	_, err := runGit("add", filePath)
	if err != nil {
		return fmt.Errorf("failed to stage file '%s': %w", filePath, err)
	}
	return nil
}

// StageAll stages all changes using git add -A
func StageAll() error {
	_, err := runGit("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

// HasFileChanges checks if a specific file has unstaged changes (i.e. differs from the index)
func HasFileChanges(filePath string) bool {
	// git diff --quiet -- <file>: exit 0 = no changes, exit 1 = has changes
	_, err := runGit("diff", "--quiet", "--", filePath)
	return err != nil
}

// IsFileModifiedOrNew returns true if the given file has uncommitted modifications
// or is an untracked new file. Works for both tracked and untracked files.
func IsFileModifiedOrNew(path string) bool {
	out, err := runGit("status", "--porcelain", "--", path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// HasStagedChanges checks if there are any staged changes ready to commit
func HasStagedChanges() bool {
	// git diff --cached --quiet: exit 0 = no staged changes, exit 1 = has staged changes
	_, err := runGit("diff", "--cached", "--quiet")
	return err != nil
}

// HasUncommittedChanges reports whether there are any uncommitted changes in the
// working tree or index (i.e. staged or unstaged modifications, additions, or deletions).
func HasUncommittedChanges() bool {
	out, err := runGit("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// Commit creates a git commit with the specified message
func Commit(message string) error {
	_, err := runGit("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// commitChanges stages all changes and commits them with a descriptive message
// It generates the commit message based on changed files
// Returns error if there are no changes to commit
func commitChanges() error {
	if err := StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	if !HasStagedChanges() {
		return fmt.Errorf("no changes to commit")
	}

	message, err := generateCommitMessage()
	if err != nil {
		message = "Update project files"
	}

	if err := Commit(message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// generateCommitMessage creates a descriptive commit message from changed files
func generateCommitMessage() (string, error) {
	stagedFiles, err := runGit("diff", "--cached", "--name-only")
	if err != nil {
		return "", fmt.Errorf("failed to get staged files: %w", err)
	}

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
	logRange := fmt.Sprintf("%s..HEAD", base)
	args := []string{"log", logRange, "--format=%h: %B"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	output, err := runGit(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get commit log: %w", err)
	}
	return output, nil
}

// BranchLogContainsPrefix reports whether any commit on branch (relative to base) has a
// commit message that contains the given prefix string. Returns false (not an error) when
// the branch does not yet exist or has no commits relative to base.
func BranchLogContainsPrefix(base, branch, prefix string) (bool, error) {
	logRange := fmt.Sprintf("%s..%s", base, branch)
	output, err := runGit("log", logRange, "--format=%s")
	if err != nil {
		return false, nil
	}

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func SwitchToBranchIfNeeded(auth *AuthConfig, branchName string) error {
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	if currentBranch != branchName {
		_ = Fetch(auth)
		if err := CheckoutOrCreateBranch(branchName); err != nil {
			return fmt.Errorf("failed to checkout review branch: %w", err)
		}
	}
	return nil
}

func CommitFileAndPush(auth *AuthConfig, filePath, branchName, commitMsg string) error {
	if err := SwitchToBranchIfNeeded(auth, branchName); err != nil {
		return err
	}

	if err := StageFile(filePath); err != nil {
		return fmt.Errorf("failed to stage project file: %w", err)
	}

	if err := Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit review findings: %w", err)
	}

	isWorkflow := auth != nil
	var owner, repo string
	if auth != nil {
		owner, repo = auth.Owner, auth.Repo
	}
	if err := PullAndPush(isWorkflow, owner, repo); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

func CommitAllAndPush(auth *AuthConfig, branchName, commitMsg string) error {
	if err := SwitchToBranchIfNeeded(auth, branchName); err != nil {
		return err
	}

	if err := StageAll(); err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}

	if err := Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit review findings: %w", err)
	}

	isWorkflow := auth != nil
	var owner, repo string
	if auth != nil {
		owner, repo = auth.Owner, auth.Repo
	}
	if err := PullAndPush(isWorkflow, owner, repo); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}
