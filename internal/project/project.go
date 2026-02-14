package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/iteration"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
)

// Execute runs the full orchestration workflow
// Steps:
// 1. Validate project file exists
// 2. Extract branch name from project file basename
// 3. Create and checkout new branch
// 4. Run iteration loop (develop + commit until complete)
// 5. Generate PR summary using AI
// 6. Push branch to origin
// 7. Create GitHub pull request
// 8. Display PR URL on success
func Execute(ctx *context.Context, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	if ctx.IsDryRun() {
		logger.Verbose("=== DRY-RUN MODE: No changes will be made ===")
	}

	// Validate project file exists
	absProjectFile, err := filepath.Abs(ctx.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	logger.Verbosef("Loading project file: %s", absProjectFile)

	// Load and validate project
	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	logger.Verbosef("Loaded project: %s", project.Name)

	// Extract branch name from project file basename
	branchName := extractBranchName(absProjectFile)
	logger.Verbosef("Branch name: %s", branchName)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseBranch := ralphConfig.BaseBranch

	// Validate git repository exists
	if !git.IsGitRepository(ctx) {
		return fmt.Errorf("not a git repository, please run 'git init' or run ralph from within a git repository")
	}

	// Check for detached HEAD state
	isDetached, err := git.IsDetachedHead(ctx)
	if err != nil {
		return fmt.Errorf("failed to check HEAD state: %w", err)
	}
	if isDetached {
		return fmt.Errorf("repository is in detached HEAD state, please checkout a branch first")
	}

	// Get current branch for potential rollback
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Verbosef("Current branch: %s", currentBranch)

	// Check if we're already on the target branch
	if currentBranch == branchName {
		logger.Verbosef("Already on branch '%s'", branchName)
	} else {
		// Check if branch exists but is not currently active
		if git.BranchExists(ctx, branchName) {
			return fmt.Errorf("branch '%s' already exists but is not currently active, please delete the branch or switch to it manually before running", branchName)
		}

		logger.Verbosef("Creating branch: %s", branchName)
		if err := git.CreateBranch(ctx, branchName); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}

		if err := git.CheckoutBranch(ctx, branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}

		logger.Successf("Created branch: %s", branchName)
	}

	// Register cleanup to return to original branch on failure
	if cleanupRegistrar != nil {
		cleanupRegistrar(func() {
			logger.Verbosef("Returning to original branch: %s", currentBranch)
			_ = git.CheckoutBranch(ctx, currentBranch)
		})
	}

	// Run iteration loop
	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations)

	iterCount, err := iteration.RunIterationLoop(ctx, cleanupRegistrar)
	if err != nil {
		// Send failure notification
		notify.Error(project.Name, ctx.ShouldNotify())
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterCount)

	// Generate PR summary using AI
	logger.Verbose("Generating PR summary...")
	prSummary, err := ai.GeneratePRSummary(ctx, absProjectFile, iterCount)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}
	logger.Verbose("PR summary generated")

	if ctx.IsVerbose() {
		logger.Verbosef("PR Summary:\n%s", prSummary)
	}

	// Push branch to origin
	logger.Verbosef("Pushing branch '%s' to origin...", branchName)
	remoteURL, err := git.PushBranch(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	logger.Verbosef("Remote URL: %s", remoteURL)

	// Check if gh CLI is available and authenticated
	if !github.IsGHInstalled(ctx) {
		return fmt.Errorf("gh CLI is not installed, please install it to create pull requests")
	}

	if !github.IsAuthenticated(ctx) {
		return fmt.Errorf("gh CLI is not authenticated, please run 'gh auth login'")
	}

	// Create GitHub pull request
	prTitle := project.Description
	if prTitle == "" {
		prTitle = project.Name
	}

	logger.Verbose("Creating GitHub pull request...")
	prURL, err := github.CreatePR(ctx, prTitle, prSummary, baseBranch, branchName)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Successf("Pull request created: %s", prURL)

	// Send success notification
	notify.Success(project.Name, ctx.ShouldNotify())

	return nil
}

// extractBranchName extracts a branch name from a project file path
// Removes the file extension and sanitizes for git branch naming
func extractBranchName(projectFile string) string {
	// Get the base filename without directory
	basename := filepath.Base(projectFile)

	// Remove file extension
	name := strings.TrimSuffix(basename, filepath.Ext(basename))

	// Sanitize for git branch naming:
	// - Replace spaces, underscores, and dots with hyphens
	// - Convert to lowercase
	// - Remove special characters except hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result.WriteRune(ch)
		}
	}

	// Remove leading/trailing hyphens and collapse multiple hyphens
	finalName := strings.Trim(result.String(), "-")

	// Collapse multiple consecutive hyphens into one
	for strings.Contains(finalName, "--") {
		finalName = strings.ReplaceAll(finalName, "--", "-")
	}

	// Ensure we have a valid name
	if finalName == "" {
		finalName = "unnamed-project"
	}

	return finalName
}
