package project

import (
	gocontext "context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
func Execute(ctx *context.Context, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	if !ctx.IsLocal() {
		return executeRemote(ctx, ctx.ProjectFile())
	}

	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project file: %w", err)
	}

	branchName := SanitizeBranchName(project.Name)
	logger.Verbosef("Branch name: %s", branchName)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, err = git.GetCurrentBranch(); err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	baseBranch := ctx.BaseBranch()

	if len(ralphConfig.Before) > 0 {
		if err := services.RunBefore(ralphConfig.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}

	if err := validateGitStateAndSwitchBranch(ctx, branchName); err != nil {
		return err
	}

	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations)

	iterCount, err := RunIterationLoop(ctx, cleanupRegistrar)
	if err != nil {
		projectName := strings.TrimSuffix(filepath.Base(absProjectFile), filepath.Ext(absProjectFile))
		notify.Error(projectName, ctx.ShouldNotify())
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterCount)

	logger.Verbose("Generating PR summary...")
	prSummary, err := ai.GeneratePRSummary(ctx, absProjectFile, iterCount, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}
	logger.Verbose("PR summary generated")

	project, err = config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration loop: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbosef("PR Summary:\n%s", prSummary)
	}

	prURL, err := createPullRequest(ctx, project, branchName, baseBranch, prSummary)
	if err != nil {
		if errors.Is(err, github.ErrNoCommitsBetweenBranches) {
			logger.Verbose("No commits ahead of base branch — all requirements were already passing; skipping PR creation")
			notify.Success(project.Name, ctx.ShouldNotify())
			return nil
		}
		return err
	}

	logger.Successf("Pull request created: %s", prURL)

	notify.Success(project.Name, ctx.ShouldNotify())

	return nil
}

func validateGitStateAndSwitchBranch(ctx *context.Context, branchName string) error {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Verbosef("Current branch: %s", currentBranch)

	if err := validateBranchSync(ctx, currentBranch); err != nil {
		return err
	}

	if currentBranch != branchName {
		if err := switchToProjectBranch(ctx, branchName); err != nil {
			return err
		}
	} else {
		logger.Verbosef("Already on branch '%s'", branchName)
	}

	return nil
}

func validateBranchSync(ctx *context.Context, currentBranch string) error {
	// Skip this check when running inside a workflow container — the container may have
	// created a fresh local branch that hasn't been pushed yet, and it will push after work is done.
	if !ctx.IsWorkflowExecution() {
		logger.Verbosef("Checking branch '%s' is in sync with remote...", currentBranch)
		if err := git.IsBranchSyncedWithRemote(currentBranch); err != nil {
			return err
		}
	} else {
		logger.Verbosef("Skipping remote sync check (running in workflow container)")
	}
	return nil
}

func switchToProjectBranch(ctx *context.Context, branchName string) error {
	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	if err := git.Fetch(auth); err != nil {
		logger.Verbosef("Could not fetch from remote (continuing anyway): %v", err)
	}

	if err := git.CheckoutOrCreateBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}
	return nil
}

func createPullRequest(ctx *context.Context, project *config.Project, branchName, baseBranch, prSummary string) (string, error) {
	// Refresh GitHub credentials immediately before creating the PR.
	// Installation tokens expire after 1 hour, so a long-running agent job may
	// have started with a valid token that is now stale. Re-running ConfigureGitAuth
	// here fetches a fresh token and re-authenticates both git and gh CLI.
	if ctx.IsWorkflowExecution() {
		owner, repoName := ctx.RepoOwnerAndName()
		if err := github.ConfigureGitAuth(gocontext.Background(), owner, repoName, github.DefaultSecretsDir); err != nil {
			return "", fmt.Errorf("failed to refresh GitHub credentials before PR creation: %w", err)
		}
	}

	if !github.IsGHReady(ctx) {
		return "", fmt.Errorf("gh CLI is not ready, please install and authenticate with 'gh auth login'")
	}

	prTitle := project.Description
	if prTitle == "" {
		prTitle = project.Name
	}

	logger.Verbose("Creating GitHub pull request...")
	prURL, err := github.CreatePR(ctx, prTitle, prSummary, baseBranch, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return prURL, nil
}

func SanitizeBranchName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")

	var result strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result.WriteRune(ch)
		}
	}

	finalName := strings.Trim(result.String(), "-")

	for strings.Contains(finalName, "--") {
		finalName = strings.ReplaceAll(finalName, "--", "-")
	}

	if finalName == "" {
		finalName = "unnamed-project"
	}

	return finalName
}

func ExtractBranchName(projectFile string) string {
	basename := filepath.Base(projectFile)

	name := strings.TrimSuffix(basename, filepath.Ext(basename))

	return SanitizeBranchName(name)
}

func executeRemote(ctx *context.Context, absProjectFile string) error {
	logger.Verbose("Submitting Argo Workflow...")

	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project file: %w", err)
	}

	projectName := project.Name
	projectBranch := SanitizeBranchName(project.Name)

	// Load configuration to get default branch
	if _, err = config.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var currentBranch string
	if ctx.Branch() != "" {
		currentBranch = ctx.Branch()
	} else {
		currentBranch, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		// Only check sync when branch is detected locally (not when pre-supplied by caller)
		logger.Verbosef("Checking branch '%s' is in sync with remote...", currentBranch)
		if err := git.IsBranchSyncedWithRemote(currentBranch); err != nil {
			return err
		}
	}

	logger.Verbose("Generating Argo Workflow...")
	wf, err := workflow.GenerateWorkflow(ctx, projectName, currentBranch, projectBranch, ctx.IsVerbose())
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	if ctx.IsVerbose() {
		workflowYAML, _ := wf.Render()
		logger.Verbosef("Generated workflow YAML:\n%s", workflowYAML)
	}

	workflowName, err := wf.Submit(wf.RalphConfig.Workflow.Namespace)
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %w", err)
	}

	logger.Successf("Workflow submitted: %s", workflowName)

	if ctx.ShouldFollow() {
		args := []string{"logs", "-n", wf.RalphConfig.Workflow.Namespace, "-f", workflowName}
		if wf.RalphConfig.Workflow.Context != "" {
			args = append(args, "--context", wf.RalphConfig.Workflow.Context)
		}
		cmd := exec.Command("argo", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			notify.Error(projectName, ctx.ShouldNotify())
			return fmt.Errorf("argo logs failed: %w", err)
		}
		notify.Success(projectName, ctx.ShouldNotify())
	} else {
		logger.Infof("To follow logs, run: argo logs -n %s -f %s", wf.RalphConfig.Workflow.Namespace, workflowName)
	}

	return nil
}
