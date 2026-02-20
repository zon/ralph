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
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

// Execute runs the full orchestration workflow
// Steps:
// 1. Validate project file exists
// 2. If remote mode: Generate and submit Argo Workflow, then exit
// 3. Run build commands once before starting iterations
// 4. Extract branch name from project file basename
// 5. Create and checkout new branch
// 6. Run iteration loop (develop + commit until complete)
// 7. Generate PR summary using AI
// 8. Push branch to origin
// 9. Create GitHub pull request
// 10. Display PR URL on success
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

	// Handle remote execution mode
	if ctx.IsRemote() {
		return executeRemote(ctx, project)
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

	iterCount, err := RunIterationLoop(ctx, cleanupRegistrar)
	if err != nil {
		// Send failure notification
		notify.Error(project.Name, ctx.ShouldNotify())
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

	// Check if project is complete
	project, err = config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration loop: %w", err)
	}

	allComplete, _, _ := config.CheckCompletion(project)

	// If project is complete, delete the project file and commit that change
	if allComplete {
		logger.Verbose("Project complete - deleting project file before creating PR")

		// Delete the project file
		if err := git.DeleteFile(ctx, absProjectFile); err != nil {
			return fmt.Errorf("failed to delete project file: %w", err)
		}

		// Commit the deletion
		commitMsg := fmt.Sprintf("Remove project file %s", filepath.Base(absProjectFile))
		if err := git.Commit(ctx, commitMsg); err != nil {
			return fmt.Errorf("failed to commit project file deletion: %w", err)
		}

		logger.Verbosef("Deleted and committed removal of project file: %s", filepath.Base(absProjectFile))
	} else {
		logger.Verbose("Project not complete - keeping project file")
	}

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

// executeRemote handles remote execution via Argo Workflows
func executeRemote(ctx *context.Context, project *config.Project) error {
	logger.Verbose("Remote execution mode enabled")

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

	// Extract project branch name from project file
	absProjectFile, err := filepath.Abs(ctx.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	projectBranch := extractBranchName(absProjectFile)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate workflow YAML - clone from current branch; workflow will create project branch inside container
	logger.Verbose("Generating Argo Workflow YAML...")
	workflowYAML, err := workflow.GenerateWorkflow(ctx, project.Name, currentBranch, projectBranch, ctx.IsDryRun(), ctx.IsVerbose())
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Note: When --dry-run is used with --remote, we submit a real workflow
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
