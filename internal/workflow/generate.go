package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
// If ctx.Repo is set (owner/repo format) the remote URL is constructed directly from it, skipping local
// git commands. If the project file path is absolute and ctx.Repo is set, it is used as-is (callers
// that bypass local git must pass a relative path).
func GenerateWorkflow(ctx *execcontext.Context, projectName, cloneBranch, projectBranch string, dryRun, verbose bool) (*Workflow, error) {
	var remoteURL string
	if ctx.Repo != "" {
		remoteURL = "https://github.com/" + ctx.Repo + ".git"
	} else {
		rawRemoteURL, err := getRemoteURL()
		if err != nil {
			return nil, fmt.Errorf("failed to get remote URL: %w", err)
		}
		remoteURL = toHTTPSURL(rawRemoteURL)
	}

	relProjectPath := ctx.ProjectFile
	if filepath.IsAbs(relProjectPath) {
		if ctx.Repo == "" {
			repoRoot, err := getRepoRoot()
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

	return GenerateWorkflowWithGitInfo(ctx, projectName, remoteURL, cloneBranch, projectBranch, relProjectPath, dryRun, verbose)
}

// GenerateWorkflowWithGitInfo builds a Workflow with provided git information.
// This allows for easier testing by accepting git info as parameters.
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, cloneBranch, projectBranch, relProjectPath string, dryRun, verbose bool) (*Workflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var instructions string
	if ctx.InstructionsMD != "" {
		instructions = ctx.InstructionsMD
	} else if data, err := os.ReadFile(filepath.Join(cwd, ".ralph", "instructions.md")); err == nil {
		instructions = string(data)
	}

	repoName, repoOwner, err := githubpkg.ParseGitHubRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	return &Workflow{
		ProjectName:   projectName,
		RepoURL:       repoURL,
		RepoOwner:     repoOwner,
		RepoName:      repoName,
		CloneBranch:   cloneBranch,
		ProjectBranch: projectBranch,
		ProjectPath:   relProjectPath,
		Instructions:  instructions,
		DryRun:        dryRun,
		Verbose:       verbose,
		RalphConfig:   ralphConfig,
	}, nil
}

// GenerateCommentWorkflowWithGitInfo builds a Workflow for a comment-triggered event.
// The container script will call `ralph comment` with the provided body and PR number.
func GenerateCommentWorkflowWithGitInfo(projectName, repoURL, cloneBranch, projectBranch, relProjectPath, commentBody, prNumber string) (*Workflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	repoName, repoOwner, err := githubpkg.ParseGitHubRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	return &Workflow{
		ProjectName:   projectName,
		RepoURL:       repoURL,
		RepoOwner:     repoOwner,
		RepoName:      repoName,
		CloneBranch:   cloneBranch,
		ProjectBranch: projectBranch,
		ProjectPath:   relProjectPath,
		CommentBody:   commentBody,
		PRNumber:      prNumber,
		RalphConfig:   ralphConfig,
	}, nil
}

// GenerateMergeWorkflow builds a MergeWorkflow, detecting git info from the local repository.
func GenerateMergeWorkflow(prBranch string) (*MergeWorkflow, error) {
	rawRemoteURL, err := getRemoteURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL: %w", err)
	}
	remoteURL := toHTTPSURL(rawRemoteURL)

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	return GenerateMergeWorkflowWithGitInfo(remoteURL, currentBranch, prBranch)
}

// GenerateMergeWorkflowWithGitInfo builds a MergeWorkflow with provided git information.
// This allows for easier testing by accepting git info as parameters.
func GenerateMergeWorkflowWithGitInfo(repoURL, cloneBranch, prBranch string) (*MergeWorkflow, error) {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	repoName, repoOwner, err := githubpkg.ParseGitHubRemoteURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository from URL: %w", err)
	}

	return &MergeWorkflow{
		RepoURL:     repoURL,
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		CloneBranch: cloneBranch,
		PRBranch:    prBranch,
		RalphConfig: ralphConfig,
	}, nil
}

// resolveImage returns the container image string from config, falling back to the default.
func resolveImage(cfg *config.RalphConfig) string {
	imageRepo := "ghcr.io/zon/ralph"
	imageTag := DefaultContainerVersion()
	if cfg.Workflow.Image.Repository != "" {
		imageRepo = cfg.Workflow.Image.Repository
	}
	if cfg.Workflow.Image.Tag != "" {
		imageTag = cfg.Workflow.Image.Tag
	}
	return fmt.Sprintf("%s:%s", imageRepo, imageTag)
}

// submitYAML submits a raw YAML string to Argo and returns the workflow name.
// This is the shared implementation used by Workflow.Submit and MergeWorkflow.Submit.
func submitYAML(workflowYAML string, ralphConfig *config.RalphConfig, namespace string) (string, error) {
	if _, err := exec.LookPath("argo"); err != nil {
		return "", fmt.Errorf("argo CLI not found - please install Argo CLI to use remote execution: https://github.com/argoproj/argo-workflows/releases")
	}

	args := []string{"submit", "-", "-n", namespace}
	if ralphConfig.Workflow.Context != "" {
		args = append(args, "--context", ralphConfig.Workflow.Context)
	}

	cmd := exec.CommandContext(context.Background(), "argo", args...)
	cmd.Stdin = strings.NewReader(workflowYAML)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to submit workflow: %w\nOutput: %s", err, string(output))
	}

	workflowName := extractWorkflowName(string(output))
	if workflowName == "" {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			workflowName = strings.TrimSpace(lines[0])
		}
	}
	return workflowName, nil
}

// extractWorkflowName extracts the workflow name from argo submit output
func extractWorkflowName(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Name:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
