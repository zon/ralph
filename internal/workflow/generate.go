package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
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
// If ctx.Repo is set (owner/repo format) the remote URL is constructed directly from it, skipping local
// git commands. If the project file path is absolute and ctx.Repo is set, it is used as-is (callers
// that bypass local git must pass a relative path).
func GenerateWorkflow(ctx *execcontext.Context, projectName, cloneBranch, projectBranch string, verbose bool) (*Workflow, error) {
	var remoteURL string
	if ctx.Repo() != "" {
		owner, name := ctx.RepoOwnerAndName()
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		repo, err := githubpkg.GetRepo(ctx.GoContext())
		if err != nil {
			return nil, fmt.Errorf("failed to get repository: %w", err)
		}
		remoteURL = repo.CloneURL()
	}

	relProjectPath := ctx.ProjectFile()
	if filepath.IsAbs(relProjectPath) {
		if ctx.Repo() == "" {
			repoRoot, err := git.FindRepoRoot()
			if err != nil {
				return nil, fmt.Errorf("failed to get repository root: %w", err)
			}
			var err2 error
			relProjectPath, err2 = filepath.Rel(repoRoot, relProjectPath)
			if err2 != nil {
				return nil, fmt.Errorf("failed to calculate relative project path: %w", err2)
			}
		}
	}

	return GenerateWorkflowWithGitInfo(ctx, projectName, remoteURL, cloneBranch, projectBranch, relProjectPath, verbose)
}

// GenerateWorkflowWithGitInfo builds a Workflow with provided git information.
// This allows for easier testing by accepting git info as parameters.
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, cloneBranch, projectBranch, relProjectPath string, verbose bool) (*Workflow, error) {
	ralphConfig, err := config.LoadConfig()
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

	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	workflowOptions := WorkflowOptions{
		Image:         MakeImage(ralphConfig.Workflow.Image.Repository, ralphConfig.Workflow.Image.Tag),
		ConfigMaps:    ralphConfig.Workflow.ConfigMaps,
		Secrets:       ralphConfig.Workflow.Secrets,
		Env:           ralphConfig.Workflow.Env,
		DefaultBranch: ralphConfig.DefaultBranch,
		Namespace:     ralphConfig.Workflow.Namespace,
		Labels:        ralphConfig.Workflow.Labels,
	}

	kubeContext := ctx.KubeContext()
	if kubeContext == "" {
		kubeContext = ralphConfig.Workflow.Context
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
		BaseBranch:    ctx.BaseBranch(),
		Image:         workflowOptions.Image,
		ConfigMaps:    workflowOptions.ConfigMaps,
		Secrets:       workflowOptions.Secrets,
		Env:           workflowOptions.Env,
		DefaultBranch: workflowOptions.DefaultBranch,
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
	}, nil
}

// GenerateReviewWorkflow builds a Workflow for a review execution.
// cloneBranch is the branch the container will clone (typically the current local branch).
func GenerateReviewWorkflow(ctx *execcontext.Context, cloneBranch string) (*Workflow, error) {
	repo, err := githubpkg.GetRepo(ctx.GoContext())
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	kubeContext := ctx.KubeContext()
	if kubeContext == "" {
		kubeContext = ralphConfig.Workflow.Context
	}

	return &Workflow{
		ProjectName:   "review",
		ProjectBranch: "review",
		Review:        true,
		Repo:          repo,
		CloneBranch:   cloneBranch,
		BaseBranch:    ctx.BaseBranch(),
		Verbose:       ctx.IsVerbose(),
		Model:         ctx.Model(),
		Image:         MakeImage(ralphConfig.Workflow.Image.Repository, ralphConfig.Workflow.Image.Tag),
		ConfigMaps:    ralphConfig.Workflow.ConfigMaps,
		Secrets:       ralphConfig.Workflow.Secrets,
		Env:           ralphConfig.Workflow.Env,
		KubeContext:   kubeContext,
		Namespace:     ralphConfig.Workflow.Namespace,
		Labels:        ralphConfig.Workflow.Labels,
	}, nil
}

// GenerateMergeWorkflow builds a MergeWorkflow, detecting git info from the local repository.
func GenerateMergeWorkflow(prBranch string) (*MergeWorkflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	repo, err := githubpkg.GetRepo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	opts := WorkflowOptions{
		Image:       MakeImage(ralphConfig.Workflow.Image.Repository, ralphConfig.Workflow.Image.Tag),
		KubeContext: ralphConfig.Workflow.Context,
		Namespace:   ralphConfig.Workflow.Namespace,
	}

	return GenerateMergeWorkflowWithGitInfo(repo.CloneURL(), currentBranch, prBranch, "", opts)
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
