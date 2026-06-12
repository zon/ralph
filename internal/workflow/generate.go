package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/version"
)

// DefaultContainerVersion returns the default container image tag read from the embedded VERSION file.
// Kept as a function for use in tests.
func DefaultContainerVersion() string {
	return version.Version()
}

// GenerateWorkflow builds a Workflow for remote execution.
// cloneBranch is the branch the container will clone (current local branch).
// projectBranch is the branch the container will create and work on (derived from the project file name).
// baseBranch is the already-resolved base branch for PR creation (see specs/features/ralph/run/spec.md).
// repoURL is the git remote URL and relProjectPath is the project file path relative to the repo root —
// both are resolved by the caller so that git and GitHub discovery are decoupled from generation logic.
func GenerateWorkflow(ctx *execcontext.Context, projectName, cloneBranch, projectBranch, baseBranch string, verbose bool, repoURL, relProjectPath string) (*Workflow, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var instructions string
	if ctx.InstructionsMD() != "" {
		instructions = ctx.InstructionsMD()
	} else if data, err := os.ReadFile(filepath.Join(cwd, ".ralph", "instructions.md")); err == nil {
		instructions = string(data)
	}

	return GenerateWorkflowWithGitInfo(ctx, projectName, repoURL, cloneBranch, projectBranch, baseBranch, relProjectPath, verbose, cfg, instructions)
}

// GenerateWorkflowWithGitInfo builds a Workflow with provided git information, config,
// and instructions. It does not perform any I/O itself — the caller supplies the loaded
// config and instructions so that test doubles can be provided.
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, cloneBranch, projectBranch, baseBranch, relProjectPath string, verbose bool, cfg *config.RalphConfig, instructions string) (*Workflow, error) {
	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	workflowOptions := WorkflowOptions{
		Image:      MakeImage(cfg.Workflow.Image.Repository, cfg.Workflow.Image.Tag),
		ConfigMaps: cfg.Workflow.ConfigMaps,
		Secrets:    cfg.Workflow.Secrets,
		Env:        cfg.Workflow.Env,
		Namespace:  cfg.Workflow.Namespace,
		Labels:     cfg.Workflow.Labels,
	}

	kubeContext := ctx.KubeContext()
	if kubeContext == "" {
		kubeContext = cfg.Workflow.Context
	}

	return &Workflow{
		ProjectName:   projectName,
		Repo:          repo,
		CloneBranch:   cloneBranch,
		ProjectBranch: projectBranch,
		ProjectPath:   relProjectPath,
		Instructions:  instructions,
		Verbose:       verbose,
		DebugBranch:   ctx.DebugBranch(),
		BaseBranch:    baseBranch,
		Image:         workflowOptions.Image,
		ConfigMaps:    workflowOptions.ConfigMaps,
		Secrets:       workflowOptions.Secrets,
		Env:           workflowOptions.Env,
		KubeContext:   kubeContext,
		Namespace:     workflowOptions.Namespace,
		NoServices:    ctx.NoServices(),
		MaxIterations: ctx.MaxIterations(),
		Model:         ctx.Model(),
		Labels:        workflowOptions.Labels,
	}, nil
}

// GenerateCommentWorkflowWithGitInfo builds a Workflow for a comment-triggered event.
// The container script will call `ralph comment` with the provided body and PR number.
func GenerateCommentWorkflowWithGitInfo(projectName, repoURL, cloneBranch, projectBranch, relProjectPath, commentBody, prNumber string, opts WorkflowOptions) (*Workflow, error) {
	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	return &Workflow{
		ProjectName:   projectName,
		Repo:          repo,
		CloneBranch:   cloneBranch,
		ProjectBranch: projectBranch,
		ProjectPath:   relProjectPath,
		CommentBody:   commentBody,
		PRNumber:      prNumber,
		Image:         opts.Image,
		KubeContext:   opts.KubeContext,
		Namespace:     opts.Namespace,
	}, nil
}

// GenerateMergeWorkflow builds a MergeWorkflow from caller-supplied git info.
// repoURL and currentBranch are resolved by the caller so that git and GitHub
// discovery are decoupled from generation logic.
func GenerateMergeWorkflow(prBranch, repoURL, currentBranch string) (*MergeWorkflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	opts := WorkflowOptions{
		Image:       MakeImage(ralphConfig.Workflow.Image.Repository, ralphConfig.Workflow.Image.Tag),
		KubeContext: ralphConfig.Workflow.Context,
		Namespace:   ralphConfig.Workflow.Namespace,
	}

	return GenerateMergeWorkflowWithGitInfo(repoURL, currentBranch, prBranch, "", opts)
}

// GenerateMergeWorkflowWithGitInfo builds a MergeWorkflow with provided git information.
// This allows for easier testing by accepting git info as parameters.
func GenerateMergeWorkflowWithGitInfo(repoURL, cloneBranch, prBranch, prNumber string, opts WorkflowOptions) (*MergeWorkflow, error) {
	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	return &MergeWorkflow{
		Repo:        repo,
		CloneBranch: cloneBranch,
		PRBranch:    prBranch,
		PRNumber:    prNumber,
		Image:       opts.Image,
		KubeContext: opts.KubeContext,
		Namespace:   opts.Namespace,
	}, nil
}

// GenerateCommandWorkflow builds a Workflow for remote command execution,
// cloning the current branch and running the supplied command.
// remoteURL is resolved by the caller so that git discovery is decoupled
// from generation logic.
func GenerateCommandWorkflow(ctx *execcontext.Context, cloneBranch, remoteURL string) (*Workflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	repo, err := githubpkg.ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}

	opts := workflowOptionsFromConfig(ralphConfig, ctx)

	return &Workflow{
		ProjectName: "command",
		Repo:        repo,
		CloneBranch: cloneBranch,
		Command:     ctx.Command(),
		Verbose:     ctx.IsVerbose(),
		DebugBranch: ctx.DebugBranch(),
		NoServices:  ctx.NoServices(),
		Model:       ctx.Model(),
		Image:       opts.Image,
		ConfigMaps:  opts.ConfigMaps,
		Secrets:     opts.Secrets,
		Env:         opts.Env,
		KubeContext: opts.KubeContext,
		Namespace:   opts.Namespace,
		Labels:      opts.Labels,
	}, nil
}

// resolveImage returns the container image string from config, falling back to the default.
func resolveImage(imageRepository, imageTag string) string {
	imageRepo := "ghcr.io/zon/ralph"
	imageVersion := DefaultContainerVersion()
	if imageRepository != "" {
		imageRepo = imageRepository
	}
	if imageTag != "" {
		imageVersion = imageTag
	}
	return fmt.Sprintf("%s:%s", imageRepo, imageVersion)
}
