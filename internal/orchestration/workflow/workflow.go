package workflow

import (
	gocontext "context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

const DefaultSecretsDir = "/secrets/github"

type GitHubAuthClient interface {
	ConfigureGitAuth(ctx gocontext.Context, owner, repo, secretsDir string) error
}

type OpenCodeClient interface {
	SetupOpenCodeCredentials(out *output.Client) error
}

type GitClient interface {
	Config(global bool, key, value string) error
	FetchBranch(out *output.Client, branch string) error
	RevParse(args ...string) (string, error)
	MergeBase(a, b string) (string, error)
	Merge(branch string) error
	AbortMerge() error
	TmpPath(name string) (string, error)
	GetCurrentBranch() (string, error)
	SanitizeBranchName(name string) string
}

type WorkspaceClient interface {
	PrepareWorkspace(out *output.Client, repoURL, branch, workDir string) error
}

type WorkspaceSetupClient interface {
	Run() error
}

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type ProjectLoader interface {
	Load(path string) (*project.Project, error)
}

type ProjectExecutionSetup struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}

type ProjectExecutor interface {
	Execute(ctx *context.Context, cleanupRegistrar func(func()), setup *ProjectExecutionSetup) error
}

type Workflow struct {
	githubAuth     GitHubAuthClient
	openCode       OpenCodeClient
	git            GitClient
	workspace      WorkspaceClient
	workspaceSetup WorkspaceSetupClient
	configLoader   ConfigLoader
	projectLoader  ProjectLoader
	executor       ProjectExecutor
}

func New(
	githubAuth GitHubAuthClient,
	openCode OpenCodeClient,
	git GitClient,
	workspace WorkspaceClient,
	workspaceSetup WorkspaceSetupClient,
	configLoader ConfigLoader,
	projectLoader ProjectLoader,
	executor ProjectExecutor,
) *Workflow {
	return &Workflow{
		githubAuth:     githubAuth,
		openCode:       openCode,
		git:            git,
		workspace:      workspace,
		workspaceSetup: workspaceSetup,
		configLoader:   configLoader,
		projectLoader:  projectLoader,
		executor:       executor,
	}
}

func (w *Workflow) Run(ctx *context.Context, cleanupRegistrar func(func())) error {
	ctx.Output().Info("Executing workflow inside container...")

	if err := w.setupGitHubAuth(ctx); err != nil {
		return fmt.Errorf("failed to setup GitHub auth: %w", err)
	}

	if err := w.openCode.SetupOpenCodeCredentials(ctx.Output()); err != nil {
		return fmt.Errorf("failed to setup OpenCode credentials: %w", err)
	}

	w.configureGitUser(ctx)

	if err := w.cloneAndSetupRepo(ctx); err != nil {
		return fmt.Errorf("failed to clone and setup repo: %w", err)
	}

	if err := w.syncBaseBranch(ctx); err != nil {
		return fmt.Errorf("failed to sync base branch: %w", err)
	}

	return w.runProject(ctx, cleanupRegistrar)
}

func (w *Workflow) setupGitHubAuth(ctx *context.Context) error {
	owner, repo := ctx.RepoOwnerAndName()
	if owner == "" || repo == "" {
		return fmt.Errorf("failed to parse owner/repo: %s", ctx.Repo())
	}

	ctx.Output().Info("Setting up GitHub App token and configuring git authentication...")
	return w.githubAuth.ConfigureGitAuth(gocontext.Background(), owner, repo, DefaultSecretsDir)
}

func (w *Workflow) configureGitUser(ctx *context.Context) {
	ctx.Output().Info("Configuring git user...")
	_ = w.git.Config(true, "user.name", ctx.BotName())
	_ = w.git.Config(true, "user.email", ctx.BotEmail())
}

func (w *Workflow) cloneAndSetupRepo(ctx *context.Context) error {
	cloneBranch := os.Getenv("GIT_BRANCH")
	if err := w.workspace.PrepareWorkspace(ctx.Output(), ctx.RepoURL(), cloneBranch, "/workspace/repo"); err != nil {
		return err
	}

	if err := w.workspaceSetup.Run(); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	return nil
}

func (w *Workflow) syncBaseBranch(ctx *context.Context) error {
	ctx.Output().Infof("Base branch: %s", ctx.BaseBranch())

	if err := w.fetchBaseBranch(ctx); err != nil {
		ctx.Output().Warnf("failed to fetch base branch: %v", err)
		return nil
	}

	needsMerge, err := w.checkIfMergeNeeded(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if merge needed: %w", err)
	}

	if !needsMerge {
		ctx.Output().Info("Project branch is up-to-date with base branch")
		return nil
	}

	ctx.Output().Info("Project branch is behind base branch, attempting merge...")
	return w.mergeBaseBranch(ctx)
}

func (w *Workflow) fetchBaseBranch(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	ctx.Output().Infof("Fetching base branch: %s", baseBranch)
	return w.git.FetchBranch(ctx.Output(), baseBranch)
}

func (w *Workflow) checkIfMergeNeeded(ctx *context.Context) (bool, error) {
	baseBranch := ctx.BaseBranch()
	if _, err := w.git.RevParse("--verify", baseBranch); err != nil {
		return false, nil
	}

	mergeBase, err := w.git.MergeBase("HEAD", baseBranch)
	if err != nil {
		return false, err
	}

	baseCommit, err := w.git.RevParse(baseBranch)
	if err != nil {
		return false, err
	}

	return mergeBase != baseCommit, nil
}

func (w *Workflow) mergeBaseBranch(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	if err := w.git.Merge(baseBranch); err != nil {
		ctx.Output().Info("Merge had conflicts - resolving with AI...")
		_ = w.git.AbortMerge()

		return w.resolveConflictsWithAI(ctx)
	}

	ctx.Output().Info("Merge successful (fast-forward or no conflicts)")
	return nil
}

func (w *Workflow) resolveConflictsWithAI(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	ctx.Output().Info("Running AI to resolve merge conflicts...")

	instructions := fmt.Sprintf(`You need to resolve merge conflicts between the base branch (%s) and the current branch (%s).

Steps:
1. Run 'git merge %s' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. Run tests to ensure the merged code is correct
4. After resolving and verifying with tests, run 'git add <resolved-files>' to stage them (the system will automatically commit)

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
`, baseBranch, ctx.Branch(), baseBranch)

	instructionsFile, err := w.git.TmpPath("merge-instructions.md")
	if err != nil {
		return fmt.Errorf("failed to get tmp path for merge instructions: %w", err)
	}

	if err := os.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write merge instructions: %w", err)
	}

	ctx.SetBaseBranch(baseBranch)

	return w.prepareAndExecute(ctx, nil, instructionsFile)
}

func (w *Workflow) runProject(ctx *context.Context, cleanupRegistrar func(func())) error {
	ctx.Output().Info("Running project...")
	return w.prepareAndExecute(ctx, cleanupRegistrar, "")
}

func (w *Workflow) prepareAndExecute(ctx *context.Context, cleanupRegistrar func(func()), instructionsFile string) error {
	projectPath := ctx.ProjectFile()
	if !filepath.IsAbs(projectPath) {
		var err error
		projectPath, err = filepath.Abs(projectPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project file path: %w", err)
		}
	}

	ralphConfig, err := w.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	maxIterations := resolveMaxIterations(ralphConfig, ctx.MaxIterations())

	ctx.SetProjectFile(projectPath)
	ctx.SetMaxIterations(maxIterations)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	if instructionsFile != "" {
		ctx.SetInstructions(instructionsFile)
	}

	proj, err := w.projectLoader.Load(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	currentBranch, err := w.git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	projectBranch := w.git.SanitizeBranchName(proj.Slug)
	setup := &ProjectExecutionSetup{
		ProjectFile:   projectPath,
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    ctx.BaseBranch(),
	}

	if err := w.executor.Execute(ctx, cleanupRegistrar, setup); err != nil {
		return fmt.Errorf("ralph execution failed: %w", err)
	}

	return nil
}

func resolveMaxIterations(cfg *config.RalphConfig, flagMaxIterations int) int {
	if flagMaxIterations > 0 {
		return flagMaxIterations
	}
	if cfg.MaxIterations > 0 {
		return cfg.MaxIterations
	}
	return 10
}
