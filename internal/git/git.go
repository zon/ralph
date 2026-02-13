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
