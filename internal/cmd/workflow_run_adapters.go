package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	orchestrationWorkflow "github.com/zon/ralph/internal/orchestration/workflow"
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
	owner, repo := a.ctx.RepoOwnerAndName()
	if owner == "" || repo == "" {
		owner, repo = parseRepo(flags.Repo)
	}
	secretsDir := github.DefaultSecretsDir
	if sd := os.Getenv("SECRETS_DIR"); sd != "" {
		secretsDir = sd
	}
	if err := github.ConfigureGitAuth(context.Background(), owner, repo, secretsDir); err != nil {
		return fmt.Errorf("failed to setup GitHub auth: %w", err)
	}
	if err := workspace.SetupOpenCodeCredentials(a.ctx.Output()); err != nil {
		return fmt.Errorf("failed to setup credentials: %w", err)
	}
	_ = git.Config(true, "user.name", flags.BotName)
	_ = git.Config(true, "user.email", flags.BotEmail)

	cloneBranch := flags.CloneBranch
	if cloneBranch == "" {
		cloneBranch = os.Getenv("GIT_BRANCH")
	}

	cloneURL := repoCloneURL(owner, repo)
	if err := workspace.PrepareWorkspace(a.ctx.Output(), cloneURL, cloneBranch, workspace.DefaultWorkDir); err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}

	if flags.TargetBranch != "" {
		if err := checkoutWorkflowBranch(flags.TargetBranch, flags.CreateBranch); err != nil {
			return err
		}
	}

	if flags.Symlinks {
		setupCmd := &SetupWorkspaceCmd{WorkspaceDir: workspace.DefaultWorkspaceDir, out: a.ctx.Output()}
		if err := setupCmd.Run(); err != nil {
			return fmt.Errorf("failed to setup workspace symlinks: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// gitAdapter
// ---------------------------------------------------------------------------

type gitAdapter struct{}

func (a *gitAdapter) FetchBranch(branch string) error {
	_, err := runGit("fetch", "origin", branch+":"+branch)
	if err != nil {
		_, err = runGit("fetch", "origin", branch)
		if err != nil {
			return fmt.Errorf("failed to fetch branch %s: %w", branch, err)
		}
	}
	return nil
}

func (a *gitAdapter) NeedsMerge(branch string) (bool, error) {
	_, err := runGit("rev-parse", "--verify", branch)
	if err != nil {
		return false, nil
	}
	mergeBase, err := runGit("merge-base", "HEAD", branch)
	if err != nil {
		return false, fmt.Errorf("failed to find merge base: %w", err)
	}
	baseCommit, err := runGit("rev-parse", branch)
	if err != nil {
		return false, fmt.Errorf("failed to get base commit: %w", err)
	}
	return strings.TrimSpace(mergeBase) != strings.TrimSpace(baseCommit), nil
}

func (a *gitAdapter) Merge(branch string) error {
	_, err := runGit("merge", branch, "--no-edit")
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}
	return nil
}

func (a *gitAdapter) AbortMerge() {
	_, _ = runGit("merge", "--abort")
}

// ---------------------------------------------------------------------------
// aiAdapter
// ---------------------------------------------------------------------------

type aiAdapter struct {
	ctx              *execcontext.Context
	cleanupRegistrar func(func())
}

func (a *aiAdapter) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	instructions := fmt.Sprintf(`You need to resolve merge conflicts between the base branch (%s) and the current branch (%s).

Steps:
1. Run 'git merge %s' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. Run tests to ensure the merged code is correct
4. After resolving and verifying with tests, run 'git add <resolved-files>' to stage them (the system will automatically commit)

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
`, baseBranch, projectBranch, baseBranch)

	instructionsFile, err := git.TmpPath("merge-instructions.md")
	if err != nil {
		return fmt.Errorf("failed to get tmp path for merge instructions: %w", err)
	}
	if err := os.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write merge instructions: %w", err)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	proj, err := project.LoadProject(a.ctx.ProjectFile())
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	projectBranchName := git.SanitizeBranchName(proj.Slug)
	setup := &ExecutionSetup{
		ProjectFile:   a.ctx.ProjectFile(),
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    projectBranchName,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
	}

	a.ctx.SetInstructions(instructionsFile)
	return Execute(a.ctx, a.cleanupRegistrar, setup)
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
	return runner.RunLocal(proj, cfg)
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

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func parseRepo(repo string) (string, string) {
	if repo == "" {
		return "", ""
	}
	parts := split2(repo, "/")
	return parts[0], parts[1]
}

func split2(s, sep string) [2]string {
	var result [2]string
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			result[0] = s[:i]
			result[1] = s[i+len(sep):]
			return result
		}
	}
	result[0] = s
	return result
}

func repoCloneURL(owner, repo string) string {
	if owner == "" || repo == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
}

func checkoutWorkflowBranch(branch string, create bool) error {
	exists := remoteBranchExists(branch)
	if exists {
		return gitCheckout(branch)
	}
	if !create {
		return wksp.ErrBranchNotFound
	}
	return gitCreateBranch(branch)
}

func remoteBranchExists(branch string) bool {
	_, err := runGit("ls-remote", "--exit-code", "--heads", "origin", branch)
	return err == nil
}

func gitCheckout(branch string) error {
	_, err := runGit("checkout", branch)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", branch, err)
	}
	return nil
}

func gitCreateBranch(branch string) error {
	_, err := runGit("checkout", "-b", branch)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", branch, err)
	}
	return nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("git %v failed: %w (output: %s)", args, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}
