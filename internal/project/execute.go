package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

// Execute runs the full orchestration workflow
// Steps:
// 1. Validate project file exists
// 2. If remote mode: load project, generate and submit Argo Workflow, then exit
// 3. Run build commands once before starting iterations
// 4. Validate current branch is in sync with remote
// 5. Extract branch name from project file basename
// 6. If PROJECT_BRANCH != current branch: fetch, then checkout remote branch or create new one
// 7. Run iteration loop (develop + commit + push until complete)
// 8. Load project file for PR title/notification
// 9. Generate PR summary using AI
// 10. Create GitHub pull request
// 11. Display PR URL on success
func Execute(ctx *context.Context, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	if ctx.IsDryRun() {
		logger.Verbose("=== DRY-RUN MODE: No changes will be made ===")
	}

	// Handle Argo Workflow submission (default when not running with --local).
	if !ctx.IsLocal() {
		return executeRemote(ctx, ctx.ProjectFile)
	}

	absProjectFile, err := filepath.Abs(ctx.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	// Extract branch name from project file basename
	branchName := extractBranchName(absProjectFile)
	logger.Verbosef("Branch name: %s", branchName)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseBranch := ralphConfig.BaseBranch

	// Run build commands before starting iteration loop
	if len(ralphConfig.Builds) > 0 {
		if err := services.RunBuilds(ralphConfig.Builds, ctx.IsDryRun()); err != nil {
			return fmt.Errorf("failed to run builds: %w", err)
		}
	}

	// Validate git repository exists
	if !git.IsGitRepository(ctx) {
		return fmt.Errorf("not a git repository, please run 'git init' or run ralph from within a git repository")
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Verbosef("Current branch: %s", currentBranch)

	// Validate current branch is in sync with remote
	// Skip this check when running inside a workflow container — the container may have
	// created a fresh local branch that hasn't been pushed yet, and it will push after work is done.
	if !ctx.IsWorkflowExecution() {
		logger.Verbosef("Checking branch '%s' is in sync with remote...", currentBranch)
		if err := git.IsBranchSyncedWithRemote(ctx, currentBranch); err != nil {
			return err
		}
	} else {
		logger.Verbosef("Skipping remote sync check (running in workflow container)")
	}

	// Switch to project branch if different from current
	if currentBranch != branchName {
		// Fetch so remote-tracking refs are up to date
		if err := git.Fetch(ctx); err != nil {
			logger.Verbosef("Could not fetch from remote (continuing anyway): %v", err)
		}

		if err := git.CheckoutOrCreateBranch(ctx, branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	} else {
		logger.Verbosef("Already on branch '%s'", branchName)
	}

	// Run iteration loop
	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations)

	iterCount, err := RunIterationLoop(ctx, cleanupRegistrar)
	if err != nil {
		projectName := strings.TrimSuffix(filepath.Base(absProjectFile), filepath.Ext(absProjectFile))
		notify.Error(projectName, ctx.ShouldNotify())
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterCount)

	// Generate PR summary using AI (before potentially deleting the project file)
	logger.Verbose("Generating PR summary...")
	prSummary, err := ai.GeneratePRSummary(ctx, absProjectFile, iterCount)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}
	logger.Verbose("PR summary generated")

	// Load project for PR title and notification
	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration loop: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbosef("PR Summary:\n%s", prSummary)
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

// executeRemote handles remote execution via Argo Workflows
func executeRemote(ctx *context.Context, absProjectFile string) error {
	logger.Verbose("Submitting Argo Workflow...")

	projectName := strings.TrimSuffix(filepath.Base(absProjectFile), filepath.Ext(absProjectFile))

	// Check if we're in a git repository
	if !git.IsGitRepository(ctx) {
		return fmt.Errorf("not in a git repository - remote execution requires git")
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Ensure the current branch is in sync with its remote before submitting the workflow
	logger.Verbosef("Checking branch '%s' is in sync with remote...", currentBranch)
	if err := git.IsBranchSyncedWithRemote(ctx, currentBranch); err != nil {
		return err
	}

	projectBranch := extractBranchName(absProjectFile)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate workflow YAML - clone from current branch; workflow will create project branch inside container
	logger.Verbose("Generating Argo Workflow YAML...")
	workflowYAML, err := workflow.GenerateWorkflow(ctx, projectName, currentBranch, projectBranch, ctx.IsDryRun(), ctx.IsVerbose())
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Note: When --dry-run is used without --local, we submit a real workflow
	// but the ralph command inside the workflow will run with --dry-run flag
	if ctx.IsVerbose() {
		logger.Verbosef("Generated workflow YAML:\n%s", workflowYAML)
	}

	// Submit workflow
	workflowName, err := workflow.SubmitWorkflow(ctx, workflowYAML, ralphConfig)
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %w", err)
	}

	logger.Successf("Workflow submitted: %s", workflowName)

	if !ctx.ShouldWatch() {
		// Determine namespace for the log command
		namespace := ralphConfig.Workflow.Namespace
		if namespace == "" {
			namespace = "default"
		}
		logger.Infof("To watch logs, run: argo logs -n %s -f %s", namespace, workflowName)
	}

	return nil
}
