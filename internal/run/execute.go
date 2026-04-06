package run

import (
	gocontext "context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

type ExecutionSetup struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}

func PrepareExecution(ctx *context.Context) (*ExecutionSetup, error) {
	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project file path: %w", err)
	}

	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load project file: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Name)
	logger.Verbosef("Branch name: %s", branchName)

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	baseBranch := ctx.BaseBranch()
	if baseBranch == "" {
		baseBranch = ralphConfig.DefaultBranch
	}

	return &ExecutionSetup{
		ProjectFile:   absProjectFile,
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    branchName,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
	}, nil
}

func Execute(ctx *context.Context, cleanupRegistrar func(func()), setup *ExecutionSetup) error {
	if !ctx.IsLocal() {
		return executeRemote(ctx, setup.ProjectFile)
	}

	if err := infrastructureRunBeforeCommands(setup.Config); err != nil {
		return err
	}

	if err := git.ValidateGitStateAndSwitchBranch(ctx, setup.BranchName); err != nil {
		return err
	}

	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations)

	iterCount, err := RunIterationLoop(ctx, cleanupRegistrar, setup.Project)
	if err != nil {
		notify.Error(setup.Project.Name, ctx.ShouldNotify())
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterCount)

	logger.Verbose("Generating PR summary...")

	commitLog, err := infrastructureGetCommitLog(setup.BaseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	allComplete, passingCount, failingCount := project.CheckCompletion(setup.Project)
	projectStatus := fmt.Sprintf("%d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	prSummary, err := ai.GeneratePRSummary(ctx, setup.Project.Description, projectStatus, setup.BaseBranch, commitLog)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}
	logger.Verbose("PR summary generated")

	setup.Project, err = project.LoadProject(setup.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration loop: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbosef("PR Summary:\n%s", prSummary)
	}

	prURL, err := github.CreatePullRequest(ctx, setup.Project, setup.BranchName, setup.BaseBranch, prSummary)
	if err != nil {
		if errors.Is(err, github.ErrNoCommitsBetweenBranches) {
			logger.Verbose("No commits ahead of base branch — all requirements were already passing; skipping PR creation")
			notify.Success(setup.Project.Name, ctx.ShouldNotify())
			return nil
		}
		return err
	}

	logger.Successf("Pull request created: %s", prURL)

	notify.Success(setup.Project.Name, ctx.ShouldNotify())

	return nil
}

func infrastructureRunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := services.RunBefore(cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}

func infrastructureGetCommitLog(baseBranch string, n int) (string, error) {
	return git.GetCommitLog(baseBranch, n)
}

func CreatePullRequest(ctx *context.Context, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
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

	if !github.IsReady() {
		return "", fmt.Errorf("gh CLI is not ready, please install and authenticate with 'gh auth login'")
	}

	prTitle := proj.Description
	if prTitle == "" {
		prTitle = proj.Name
	}

	logger.Verbose("Creating GitHub pull request...")
	prURL, err := github.CreatePR(prTitle, prSummary, baseBranch, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return prURL, nil
}

func executeRemote(ctx *context.Context, absProjectFile string) error {
	logger.Verbose("Submitting Argo Workflow...")

	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project file: %w", err)
	}

	projectName := proj.Name
	projectBranch := git.SanitizeBranchName(proj.Name)

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

	workflowName, err := wf.Submit()
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %w", err)
	}

	logger.Successf("Workflow submitted: %s", workflowName)

	if ctx.ShouldFollow() {
		if err := argo.FollowLogs(wf.Namespace, workflowName, wf.KubeContext); err != nil {
			notify.Error(projectName, ctx.ShouldNotify())
			return fmt.Errorf("argo logs failed: %w", err)
		}
		notify.Success(projectName, ctx.ShouldNotify())
	} else {
		logger.Infof("To follow logs, run: argo logs -n %s %s -f", wf.Namespace, workflowName)
	}

	return nil
}
