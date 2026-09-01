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
// baseBranch is the already-resolved base branch for PR creation (see specs/ralph/run.md).
// items is the already-resolved item query selecting the item array, and cleanup reports whether the
// project file should be deleted once every item is complete. Both are resolved by the caller so the
// workflow container does not re-resolve them from config.
// repoURL is the git remote URL and relProjectPath is the project file path relative to the repo root.
// Both are resolved by the caller so that git and GitHub discovery are decoupled from generation logic.
func GenerateWorkflow(ctx *execcontext.Context, projectName, cloneBranch, projectBranch, baseBranch, items string, cleanup bool, verbose bool, repoURL, relProjectPath string) (*Workflow, error) {
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

	return GenerateWorkflowWithGitInfo(ctx, projectName, repoURL, cloneBranch, projectBranch, baseBranch, items, cleanup, relProjectPath, verbose, cfg, instructions)
}

// GenerateWorkflowWithGitInfo builds a Workflow with provided git information, config,
// and instructions. It does not perform any I/O itself. The caller supplies the loaded
// config and instructions so that test doubles can be provided.
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, cloneBranch, projectBranch, baseBranch, items string, cleanup bool, relProjectPath string, verbose bool, cfg *config.RalphConfig, instructions string) (*Workflow, error) {
	repo, err := githubpkg.ParseRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	workflowOptions := workflowOptionsFromConfig(cfg, ctx)

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
		Items:         items,
		Cleanup:       cleanup,
		Image:         workflowOptions.Image,
		ConfigMaps:    workflowOptions.ConfigMaps,
		Secrets:       workflowOptions.Secrets,
		Env:           workflowOptions.Env,
		KubeContext:   workflowOptions.KubeContext,
		Namespace:     workflowOptions.Namespace,
		NoServices:    ctx.NoServices(),
		Model:         ctx.Model(),
		Agent:         ctx.Agent(),
		Variant:       ctx.Variant(),
		Labels:        workflowOptions.Labels,
		Resources:     workflowOptions.Resources,
	}, nil
}

// GenerateLoopWorkflow builds a Workflow that runs the ralph loop inside the
// container. cloneBranch is the branch the container clones (the current local
// branch) and remoteURL is the git remote URL. Both are resolved by the caller so
// that git and GitHub discovery are decoupled from generation logic. The slug,
// steps, and max iterations are carried into the container as CLI arguments to
// `ralph workflow loop`.
func GenerateLoopWorkflow(ctx *execcontext.Context, slug string, steps []string, max int, cloneBranch, remoteURL string) (*Workflow, error) {
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
		ProjectName: "loop",
		Repo:        repo,
		CloneBranch: cloneBranch,
		Loop:        &LoopSpec{Slug: slug, Steps: steps, Max: max},
		Image:       opts.Image,
		ConfigMaps:  opts.ConfigMaps,
		Secrets:     opts.Secrets,
		Env:         opts.Env,
		KubeContext: opts.KubeContext,
		Namespace:   opts.Namespace,
		NoServices:  ctx.NoServices(),
		Model:       ctx.Model(),
		Agent:       ctx.Agent(),
		Variant:     ctx.Variant(),
		Labels:      opts.Labels,
		Resources:   opts.Resources,
		Verbose:     ctx.IsVerbose(),
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
		Agent:       ctx.Agent(),
		Image:       opts.Image,
		ConfigMaps:  opts.ConfigMaps,
		Secrets:     opts.Secrets,
		Env:         opts.Env,
		KubeContext: opts.KubeContext,
		Namespace:   opts.Namespace,
		Labels:      opts.Labels,
		Resources:   opts.Resources,
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
