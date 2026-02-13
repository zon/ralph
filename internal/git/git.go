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
