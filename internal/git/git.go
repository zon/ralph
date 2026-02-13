package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
)

// GetCurrentBranch returns the name of the current git branch
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

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Current branch: %s", branch))
	}

	return branch, nil
}

// BranchExists checks if a git branch exists (local or remote)
func BranchExists(ctx *context.Context, name string) bool {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would check if branch '%s' exists", name))
		return false
	}

	// Check local branches
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", name)
	if err := cmd.Run(); err == nil {
		if ctx.IsVerbose() {
			logger.Info(fmt.Sprintf("Branch '%s' exists locally", name))
		}
		return true
	}

	// Check remote branches
	cmd = exec.Command("git", "rev-parse", "--verify", "--quiet", "origin/"+name)
	if err := cmd.Run(); err == nil {
		if ctx.IsVerbose() {
			logger.Info(fmt.Sprintf("Branch '%s' exists remotely", name))
		}
		return true
	}

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Branch '%s' does not exist", name))
	}

	return false
}

// CreateBranch creates a new git branch
func CreateBranch(ctx *context.Context, name string) error {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would create branch: %s", name))
		return nil
	}

	cmd := exec.Command("git", "branch", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w (output: %s)", name, err, out.String())
	}

	logger.Success(fmt.Sprintf("Created branch: %s", name))
	return nil
}

// CheckoutBranch switches to the specified git branch
func CheckoutBranch(ctx *context.Context, name string) error {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would checkout branch: %s", name))
		return nil
	}

	cmd := exec.Command("git", "checkout", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w (output: %s)", name, err, out.String())
	}

	logger.Success(fmt.Sprintf("Checked out branch: %s", name))
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
		logger.Info(fmt.Sprintf("[DRY-RUN] Would push branch '%s' to origin", branch))
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

	logger.Success(fmt.Sprintf("Pushed branch '%s' to origin", branch))

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
		logger.Info(fmt.Sprintf("Remote URL: %s", remoteURL))
	}

	return remoteURL, nil
}

// GetRecentCommits retrieves the last N commit messages
func GetRecentCommits(ctx *context.Context, count int) ([]string, error) {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would get last %d commits", count))
		dryRunCommits := make([]string, count)
		for i := 0; i < count; i++ {
			dryRunCommits[i] = fmt.Sprintf("dry-run commit %d", i+1)
		}
		return dryRunCommits, nil
	}

	// Use git log with format to get commit messages
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%h %s")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w (output: %s)", err, out.String())
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return []string{}, nil
	}

	commits := strings.Split(output, "\n")
	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Retrieved %d commits", len(commits)))
	}

	return commits, nil
}

// GetCommitsSince retrieves commit messages since the specified base branch
func GetCommitsSince(ctx *context.Context, base string) ([]string, error) {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would get commits since '%s'", base))
		return []string{
			"dry-run commit 1 - feature implementation",
			"dry-run commit 2 - bug fix",
			"dry-run commit 3 - tests",
		}, nil
	}

	// Use git log to get commits in current branch but not in base
	cmd := exec.Command("git", "log", fmt.Sprintf("%s..HEAD", base), "--pretty=format:%h %s")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get commits since '%s': %w (output: %s)", base, err, out.String())
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return []string{}, nil
	}

	commits := strings.Split(output, "\n")
	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Retrieved %d commits since '%s'", len(commits), base))
	}

	return commits, nil
}

// GetDiffSince returns the diff between the base branch and HEAD
func GetDiffSince(ctx *context.Context, base string) (string, error) {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would get diff since '%s'", base))
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
		logger.Info(fmt.Sprintf("Retrieved diff since '%s' (%d bytes)", base, len(diff)))
	}

	return diff, nil
}

// StageFile stages a specific file using git add
func StageFile(ctx *context.Context, filePath string) error {
	if ctx.IsDryRun() {
		logger.Info(fmt.Sprintf("[DRY-RUN] Would stage file: %s", filePath))
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
		logger.Info(fmt.Sprintf("Staged file: %s", filePath))
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
		logger.Info(fmt.Sprintf("[DRY-RUN] Would commit with message: %s", message))
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
		logger.Info(fmt.Sprintf("Committed: %s", message))
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
		logger.Warning("Failed to generate commit message: %v, using fallback", err)
		message = "Update project files"
	}

	// Commit the staged changes
	if err := Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	logger.Success("Changes committed: %s", message)
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
		logger.Info("Generating commit message for %d file(s)", fileCount)
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
