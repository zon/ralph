package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
)

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository(ctx *context.Context) bool {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check if directory is a git repository")
		return true
	}

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		if ctx.IsVerbose() {
			logger.Info("Not a git repository")
		}
		return false
	}

	if ctx.IsVerbose() {
		logger.Info("Git repository detected")
	}
	return true
}

// IsDetachedHead checks if the repository is in a detached HEAD state
func IsDetachedHead(ctx *context.Context) (bool, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check for detached HEAD state")
		return false, nil
	}

	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	err := cmd.Run()

	// Exit code 0 = on a branch (not detached)
	// Exit code 1 = detached HEAD
	isDetached := err != nil

	if ctx.IsVerbose() {
		if isDetached {
			logger.Info("Repository is in detached HEAD state")
		} else {
			logger.Info("Repository is on a branch (not detached)")
		}
	}

	return isDetached, nil
}

// GetCurrentBranch returns the name of the current git branch
// Returns error if in detached HEAD state
func GetCurrentBranch(ctx *context.Context) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would get current git branch")
		return "dry-run-branch", nil
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w (output: %s)", err, out.String())
	}

	branch := strings.TrimSpace(out.String())
	if branch == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	// Check for detached HEAD state
	if branch == "HEAD" {
		return "", fmt.Errorf("repository is in detached HEAD state, please checkout a branch first")
	}

	logger.Verbosef("Current branch: %s", branch)

	return branch, nil
}

// BranchExists checks if a git branch exists (local or remote)
func BranchExists(ctx *context.Context, name string) bool {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would check if branch '%s' exists", name)
		return false
	}

	// Check local branches
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", name)
	if err := cmd.Run(); err == nil {
		if ctx.IsVerbose() {
			logger.Infof("Branch '%s' exists locally", name)
		}
		return true
	}

	// Check remote branches
	cmd = exec.Command("git", "rev-parse", "--verify", "--quiet", "origin/"+name)
	if err := cmd.Run(); err == nil {
		if ctx.IsVerbose() {
			logger.Infof("Branch '%s' exists remotely", name)
		}
		return true
	}

	if ctx.IsVerbose() {
		logger.Infof("Branch '%s' does not exist", name)
	}

	return false
}

// RemoteBranchExists checks if a branch exists on the remote
func RemoteBranchExists(ctx *context.Context, name string) bool {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would check if branch '%s' exists on remote", name)
		return false
	}

	// Use git ls-remote to check if branch exists on remote
	// This works even if we haven't fetched recently
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if ctx.IsVerbose() {
			logger.Infof("Failed to check remote branch '%s': %v", name, err)
		}
		return false
	}

	// If output is empty, branch doesn't exist on remote
	output := strings.TrimSpace(out.String())
	exists := output != ""

	if ctx.IsVerbose() {
		if exists {
			logger.Infof("Branch '%s' exists on remote", name)
		} else {
			logger.Infof("Branch '%s' does not exist on remote", name)
		}
	}

	return exists
}

// CreateBranch creates a new git branch
func CreateBranch(ctx *context.Context, name string) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would create branch: %s", name)
		return nil
	}

	cmd := exec.Command("git", "branch", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w (output: %s)", name, err, out.String())
	}

	logger.Verbosef("Created branch: %s", name)
	return nil
}

// CheckoutBranch switches to the specified git branch
func CheckoutBranch(ctx *context.Context, name string) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would checkout branch: %s", name)
		return nil
	}

	cmd := exec.Command("git", "checkout", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w (output: %s)", name, err, out.String())
	}

	logger.Verbosef("Checked out branch: %s", name)
	return nil
}

// HasCommits checks if the current branch has any commits
func HasCommits(ctx *context.Context) bool {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check if branch has commits")
		return true
	}

	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	if err := cmd.Run(); err != nil {
		if ctx.IsVerbose() {
			logger.Info("No commits found on current branch")
		}
		return false
	}

	if ctx.IsVerbose() {
		logger.Info("Branch has commits")
	}
	return true
}

// PushBranch pushes the specified branch to origin and returns the remote URL
func PushBranch(ctx *context.Context, branch string) (string, error) {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would push branch '%s' to origin", branch)
		return "https://github.com/dry-run/repo", nil
	}

	// Check if there are commits to push
	if !HasCommits(ctx) {
		return "", fmt.Errorf("no commits to push on branch '%s'", branch)
	}

	// Push the branch with --set-upstream
	cmd := exec.Command("git", "push", "--set-upstream", "origin", branch)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to push branch '%s': %w (output: %s)", branch, err, out.String())
	}

	logger.Verbosef("Pushed branch '%s' to origin", branch)

	// Get the remote URL
	cmd = exec.Command("git", "config", "--get", "remote.origin.url")
	var urlOut bytes.Buffer
	cmd.Stdout = &urlOut
	cmd.Stderr = &urlOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w (output: %s)", err, urlOut.String())
	}

	remoteURL := strings.TrimSpace(urlOut.String())
	if ctx.IsVerbose() {
		logger.Infof("Remote URL: %s", remoteURL)
	}

	return remoteURL, nil
}

// GetCommitLog retrieves commit log formatted exactly like the reference implementation.
// Returns a single string with commits formatted as "%h: %B" (hash: full message).
// Gets all commits since base..HEAD.
func GetCommitLog(ctx *context.Context, base string) (string, error) {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would get commits since '%s'", base)
		return "abc123: dry-run commit 1 - feature implementation\ndef456: dry-run commit 2 - bug fix\nghi789: dry-run commit 3 - tests", nil
	}

	// Use git log with format matching reference: %h: %B (hash: full message body)
	logRange := fmt.Sprintf("%s..HEAD", base)
	cmd := exec.Command("git", "log", logRange, "--format=%h: %B")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit log: %w (output: %s)", err, out.String())
	}

	output := strings.TrimSpace(out.String())
	return output, nil
}

// GetDiffSince returns the diff between the base branch and HEAD
func GetDiffSince(ctx *context.Context, base string) (string, error) {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would get diff since '%s'", base)
		return "dry-run diff output:\n+added line\n-removed line", nil
	}

	// Use git diff to get changes between base and HEAD
	cmd := exec.Command("git", "diff", fmt.Sprintf("%s..HEAD", base))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get diff since '%s': %w (output: %s)", base, err, out.String())
	}

	diff := out.String()
	if ctx.IsVerbose() {
		logger.Infof("Retrieved diff since '%s' (%d bytes)", base, len(diff))
	}

	return diff, nil
}

// StageFile stages a specific file using git add
func StageFile(ctx *context.Context, filePath string) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would stage file: %s", filePath)
		return nil
	}

	cmd := exec.Command("git", "add", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file '%s': %w (output: %s)", filePath, err, out.String())
	}

	if ctx.IsVerbose() {
		logger.Infof("Staged file: %s", filePath)
	}

	return nil
}

// StageAll stages all changes using git add -A
func StageAll(ctx *context.Context) error {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would stage all changes (git add -A)")
		return nil
	}

	cmd := exec.Command("git", "add", "-A")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes: %w (output: %s)", err, out.String())
	}

	if ctx.IsVerbose() {
		logger.Info("Staged all changes")
	}

	return nil
}

// HasStagedChanges checks if there are any staged changes ready to commit
func HasStagedChanges(ctx *context.Context) bool {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check for staged changes")
		return true
	}

	// Use git diff --cached --quiet to check for staged changes
	// Exit code 0 = no staged changes, exit code 1 = has staged changes
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	hasStagedChanges := err != nil // Non-zero exit = has changes

	if ctx.IsVerbose() {
		if hasStagedChanges {
			logger.Info("Staged changes detected")
		} else {
			logger.Info("No staged changes found")
		}
	}

	return hasStagedChanges
}

// Commit creates a git commit with the specified message
func Commit(ctx *context.Context, message string) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would commit with message: %s", message)
		return nil
	}

	cmd := exec.Command("git", "commit", "-m", message)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w (output: %s)", err, out.String())
	}

	if ctx.IsVerbose() {
		logger.Infof("Committed: %s", message)
	}

	return nil
}

// CommitChanges stages all changes and commits them with a descriptive message
// It generates the commit message based on changed files
// Returns error if there are no changes to commit
func CommitChanges(ctx *context.Context) error {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would stage and commit all changes with generated message")
		return nil
	}

	// Stage all changes
	if err := StageAll(ctx); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are any staged changes
	if !HasStagedChanges(ctx) {
		return fmt.Errorf("no changes to commit")
	}

	// Get list of changed files to generate commit message
	message, err := generateCommitMessage(ctx)
	if err != nil {
		// Fallback to generic message if generation fails
		logger.Warningf("Failed to generate commit message: %v, using fallback", err)
		message = "Update project files"
	}

	// Commit the staged changes
	if err := Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	logger.Successf("Changes committed: %s", message)
	return nil
}

// generateCommitMessage creates a descriptive commit message from changed files
func generateCommitMessage(ctx *context.Context) (string, error) {
	// Get list of staged files
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

	if ctx.IsVerbose() {
		logger.Infof("Generating commit message for %d file(s)", fileCount)
	}

	// Generate message based on files changed
	if fileCount == 1 {
		return fmt.Sprintf("Update %s", files[0]), nil
	} else if fileCount <= 3 {
		// List all files if 2-3 files
		return fmt.Sprintf("Update %s", strings.Join(files, ", ")), nil
	} else {
		// For many files, use a summary
		// Try to categorize by directory or file type
		categories := categorizeFiles(files)
		if len(categories) == 1 {
			for category := range categories {
				return fmt.Sprintf("Update %s files (%d files)", category, fileCount), nil
			}
		}
		return fmt.Sprintf("Update %d files across project", fileCount), nil
	}
}

// categorizeFiles groups files by directory or type
func categorizeFiles(files []string) map[string]int {
	categories := make(map[string]int)

	for _, file := range files {
		// Try to extract directory or file type
		if strings.Contains(file, "/") {
			parts := strings.Split(file, "/")
			if len(parts) > 1 {
				dir := parts[0]
				categories[dir]++
			}
		} else {
			// Root level files - categorize by extension
			if strings.Contains(file, ".") {
				parts := strings.Split(file, ".")
				ext := parts[len(parts)-1]
				categories[ext]++
			} else {
				categories["root"]++
			}
		}
	}

	return categories
}
