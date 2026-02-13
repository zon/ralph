package run

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
func Execute(ctx *context.Context, projectFile string, maxIterations int, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	if ctx.IsDryRun() {
		logger.Info("=== DRY-RUN MODE: No changes will be made ===")
	}

	// Validate project file exists
	absProjectFile, err := filepath.Abs(projectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	logger.Info("Loading project file: %s", absProjectFile)

	// Load and validate project
	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	logger.Success("Loaded project: %s", project.Name)

	// Extract branch name from project file basename
	branchName := extractBranchName(absProjectFile)
	logger.Info("Branch name: %s", branchName)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseBranch := ralphConfig.BaseBranch

	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Full orchestration plan:")
		logger.Info("[DRY-RUN]   1. Validate git repository exists")
		logger.Info("[DRY-RUN]   2. Check for detached HEAD state")
		logger.Info("[DRY-RUN]   3. Create branch: %s", branchName)
		logger.Info("[DRY-RUN]   4. Run up to %d iterations (develop -> commit -> check)", maxIterations)
		logger.Info("[DRY-RUN]   5. Generate PR summary using AI")
		logger.Info("[DRY-RUN]   6. Push branch to origin")
		logger.Info("[DRY-RUN]   7. Create GitHub PR (base: %s, head: %s)", baseBranch, branchName)
		logger.Info("[DRY-RUN]   8. Display PR URL")
		return nil
	}

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

	// Check if branch already exists
	if git.BranchExists(ctx, branchName) {
		return fmt.Errorf("branch '%s' already exists, please delete or rename it first", branchName)
	}

	// Get current branch for potential rollback
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Info("Current branch: %s", currentBranch)

	// Create new branch
	logger.Info("Creating branch: %s", branchName)
	if err := git.CreateBranch(ctx, branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout new branch
	logger.Info("Checking out branch: %s", branchName)
	if err := git.CheckoutBranch(ctx, branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	// Register cleanup to return to original branch on failure
	if cleanupRegistrar != nil {
		cleanupRegistrar(func() {
			logger.Info("Returning to original branch: %s", currentBranch)
			_ = git.CheckoutBranch(ctx, currentBranch)
		})
	}

	// Run iteration loop
	logger.Info("Starting iteration loop (max: %d)", maxIterations)
	logger.Info("==========================================")

	iterCount, err := iteration.RunIterationLoop(ctx, absProjectFile, maxIterations, cleanupRegistrar)
	if err != nil {
		// Send failure notification
		notify.Error(project.Name, ctx.ShouldNotify())
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	logger.Info("==========================================")
	logger.Success("Iteration loop completed after %d iteration(s)", iterCount)

	// Generate PR summary using AI
	logger.Info("Generating PR summary...")
	prSummary, err := ai.GeneratePRSummary(ctx, absProjectFile, iterCount)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}
	logger.Success("PR summary generated")

	if ctx.IsVerbose() {
		logger.Info("PR Summary:\n%s", prSummary)
	}

	// Push branch to origin
	logger.Info("Pushing branch '%s' to origin...", branchName)
	remoteURL, err := git.PushBranch(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Info("Remote URL: %s", remoteURL)
	}

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

	logger.Info("Creating GitHub pull request...")
	prURL, err := github.CreatePR(ctx, prTitle, prSummary, baseBranch, branchName)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	// Display PR URL
	logger.Success("==========================================")
	logger.Success("Pull Request Created Successfully!")
	logger.Success("==========================================")
	logger.Success("URL: %s", prURL)
	logger.Success("==========================================")

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
