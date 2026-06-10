package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/opencode"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	orchestrationWorkflow "github.com/zon/ralph/internal/orchestration/workflowrun"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workspace"
)

func newOrchestrationWorkflowRunCmd(ctx *execcontext.Context, cleanupRegistrar func(func())) *orchestrationWorkflow.WorkflowRunCmd {
	return orchestrationWorkflow.NewWorkflowRunCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&gitAdapter{},
		&aiAdapter{ctx: ctx, cleanupRegistrar: cleanupRegistrar},
		&runnerAdapter{ctx: ctx, baseBranch: ctx.BaseBranch()},
		&configOptionalAdapter{},
		&projectLoadAdapter{},
		&debugAdapter{ctx: ctx},
	)
}

// ---------------------------------------------------------------------------
// workspaceSetupAdapter
// ---------------------------------------------------------------------------

type workspaceSetupAdapter struct {
	ctx *execcontext.Context
}

func (a *workspaceSetupAdapter) Setup(flags wksp.WorkspaceFlags) error {
	return wksp.New(
		&workspaceGitHubClient{ctx: a.ctx},
		&workspaceWorkspaceClient{ctx: a.ctx},
		&workspaceGitClient{ctx: a.ctx},
	).Setup(flags)
}

// ---------------------------------------------------------------------------
// workspaceGitHubClient implements orchestration/workspace.GitHubClient
// ---------------------------------------------------------------------------

type workspaceGitHubClient struct {
	ctx *execcontext.Context
}

func (c *workspaceGitHubClient) ConfigureAuth(repo string) error {
	owner, repoName := c.ctx.RepoOwnerAndName()
	if owner == "" || repoName == "" {
		owner, repoName = github.ParseRepo(repo)
	}
	secretsDir := github.DefaultSecretsDir
	if sd := os.Getenv("SECRETS_DIR"); sd != "" {
		secretsDir = sd
	}
	return github.ConfigureGitAuth(context.Background(), owner, repoName, secretsDir)
}

// ---------------------------------------------------------------------------
// workspaceWorkspaceClient implements orchestration/workspace.WorkspaceClient
// ---------------------------------------------------------------------------

type workspaceWorkspaceClient struct {
	ctx *execcontext.Context
}

func (c *workspaceWorkspaceClient) SetupCredentials() error {
	return workspace.SetupOpenCodeCredentials(c.ctx.Output())
}

func (c *workspaceWorkspaceClient) SetupSymlinks() error {
	return workspace.SetupSymlinks(c.ctx.Output())
}

// ---------------------------------------------------------------------------
// workspaceGitClient implements orchestration/workspace.GitClient
// ---------------------------------------------------------------------------

type workspaceGitClient struct {
	ctx *execcontext.Context
}

func (c *workspaceGitClient) ConfigureUser(name, email string) {
	_ = git.Config(true, "user.name", name)
	_ = git.Config(true, "user.email", email)
}

func (c *workspaceGitClient) Clone(branch string) error {
	owner, repo := c.ctx.RepoOwnerAndName()
	cloneBranch := branch
	if cloneBranch == "" {
		cloneBranch = os.Getenv("GIT_BRANCH")
	}
	cloneURL := github.CloneURL(owner, repo)
	return workspace.PrepareWorkspace(c.ctx.Output(), cloneURL, cloneBranch, workspace.DefaultWorkDir)
}

func (c *workspaceGitClient) RemoteBranchExists(branch string) (bool, error) {
	return git.RemoteBranchExists(branch)
}

func (c *workspaceGitClient) FetchAndCheckout(branch string) error {
	return git.CheckoutBranch(branch)
}

func (c *workspaceGitClient) CreateAndCheckout(branch string) error {
	return git.CreateBranch(branch)
}

// ---------------------------------------------------------------------------
// gitAdapter
// ---------------------------------------------------------------------------

type gitAdapter struct{}

func (a *gitAdapter) FetchBranch(branch string) error {
	return git.FetchBranch(branch)
}

func (a *gitAdapter) NeedsMerge(branch string) (bool, error) {
	return git.NeedsMerge(branch)
}

func (a *gitAdapter) Merge(branch string) error {
	return git.Merge(branch)
}

func (a *gitAdapter) AbortMerge() {
	_ = git.AbortMerge()
}

// ---------------------------------------------------------------------------
// aiAdapter
// ---------------------------------------------------------------------------

type aiAdapter struct {
	ctx              *execcontext.Context
	cleanupRegistrar func(func())
}

func (a *aiAdapter) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	prompt, err := ai.BuildResolveMergeConflictsPrompt(baseBranch, projectBranch)
	if err != nil {
		return err
	}
	return ai.RunAgent(a.ctx, opencode.New(), prompt)
}

// ---------------------------------------------------------------------------
// runnerAdapter
// ---------------------------------------------------------------------------

type runnerAdapter struct {
	ctx        *execcontext.Context
	baseBranch string
}

func (a *runnerAdapter) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	runner := NewLocalRunner(a.ctx, a.baseBranch)
	return runner.RunLocal(project.ForProjectInput(proj), cfg)
}

// ---------------------------------------------------------------------------
// configOptionalAdapter
// ---------------------------------------------------------------------------

type configOptionalAdapter struct{}

func (a *configOptionalAdapter) LoadOptional() (*config.RalphConfig, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return &config.RalphConfig{}, nil
	}
	return cfg, nil
}

// ---------------------------------------------------------------------------
// projectLoadAdapter
// ---------------------------------------------------------------------------

type projectLoadAdapter struct{}

func (a *projectLoadAdapter) Load(path string) (*project.Project, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project file path: %w", err)
	}
	return project.LoadProject(absPath)
}

// ---------------------------------------------------------------------------
// debugAdapter
// ---------------------------------------------------------------------------

type debugAdapter struct {
	ctx *execcontext.Context
}

func (a *debugAdapter) Setup(branch string) error {
	a.ctx.SetDebugBranch(branch)
	return nil
}


