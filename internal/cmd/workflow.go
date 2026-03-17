package cmd

import (
	gocontext "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zon/ralph/internal/cleanup"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

const (
	DefaultSecretsDir         = "/secrets/github"
	DefaultOpenCodeSecretsDir = "/secrets/opencode"
)

type WorkflowCmd struct {
	Repo           string `arg:"" help:"GitHub repository (owner/repo)" required:""`
	ProjectPath    string `arg:"" help:"Path to project YAML file within the repository" required:""`
	ProjectBranch  string `help:"Branch to clone or create" name:"project-branch" required:""`
	BaseBranch     string `help:"Base branch for PR creation" name:"base" short:"B" required:""`
	BotName        string `help:"Git user name for commits" default:"zalphen[bot]"`
	BotEmail       string `help:"Git user email for commits" default:"zalphen[bot]@users.noreply.github.com"`
	DebugBranch    string `help:"Ralph branch to use for debug mode (clones ralph from this branch and runs via go run)" name:"debug" optional:""`
	Verbose        bool   `help:"Enable verbose logging" default:"false"`
	NoServices     bool   `help:"Skip service startup" default:"false"`
	InstructionsMD string `help:"Inline instructions content" name:"instructions-md" optional:""`
	MaxIterations  int    `help:"Maximum number of iterations" name:"max-iterations" default:"0"`
	Model          string `help:"Override the AI model from config" name:"model" optional:""`

	cleanupRegistrar cleanup.Registrar `kong:"-"`
}

func (w *WorkflowCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(w.Verbose)
	ctx.SetNoServices(w.NoServices)
	ctx.SetRepo(w.Repo)
	ctx.SetBranch(w.ProjectBranch)
	ctx.SetBaseBranch(w.BaseBranch)
	ctx.SetProjectFile(w.ProjectPath)
	ctx.SetInstructionsMD(w.InstructionsMD)
	ctx.SetDebugBranch(w.DebugBranch)
	ctx.SetMaxIterations(w.MaxIterations)
	ctx.SetBotName(w.BotName)
	ctx.SetBotEmail(w.BotEmail)
	ctx.SetModel(w.Model)

	logger.Info("Executing workflow inside container...")

	if err := w.setupGitHubAuth(ctx); err != nil {
		return fmt.Errorf("failed to setup GitHub auth: %w", err)
	}

	if err := w.setupOpenCodeCredentials(); err != nil {
		return fmt.Errorf("failed to setup OpenCode credentials: %w", err)
	}

	w.configureGitUser(ctx)

	if err := w.cloneAndSetupRepo(ctx); err != nil {
		return fmt.Errorf("failed to clone and setup repo: %w", err)
	}

	if err := w.syncBaseBranch(ctx); err != nil {
		return fmt.Errorf("failed to sync base branch: %w", err)
	}

	if err := w.runProject(ctx); err != nil {
		return fmt.Errorf("failed to run project: %w", err)
	}

	w.displayStats()

	return nil
}

func (w *WorkflowCmd) setupGitHubAuth(ctx *context.Context) error {
	owner, repo := ctx.RepoOwnerAndName()
	if owner == "" || repo == "" {
		return fmt.Errorf("failed to parse owner/repo: %s", ctx.Repo())
	}

	logger.Info("Setting up GitHub App token and configuring git authentication...")
	return github.ConfigureGitAuth(gocontext.Background(), owner, repo, DefaultSecretsDir)
}

func (w *WorkflowCmd) setupOpenCodeCredentials() error {
	logger.Info("Setting up OpenCode credentials...")

	openCodeDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode")
	if err := os.MkdirAll(openCodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode directory: %w", err)
	}

	authFile := filepath.Join(DefaultOpenCodeSecretsDir, "auth.json")
	if _, err := os.Stat(authFile); err == nil {
		destPath := filepath.Join(openCodeDir, "auth.json")
		data, err := os.ReadFile(authFile)
		if err != nil {
			return fmt.Errorf("failed to read auth file: %w", err)
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write auth file: %w", err)
		}
		logger.Infof("Copied OpenCode credentials to %s", destPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check auth file: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) configureGitUser(ctx *context.Context) {
	logger.Info("Configuring git user...")
	_ = git.Config(true, "user.name", ctx.BotName())
	_ = git.Config(true, "user.email", ctx.BotEmail())
}

func (w *WorkflowCmd) cloneAndSetupRepo(ctx *context.Context) error {
	logger.Infof("Cloning repository: %s", ctx.RepoURL())

	workDir := "/workspace/repo"
	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}

	if _, err := os.Stat(workDir); err == nil {
		os.RemoveAll(workDir)
	}

	// Use GIT_BRANCH env var for cloning (set by the workflow template to the
	// original local branch), not ctx.Branch() which holds the project branch.
	cloneBranch := os.Getenv("GIT_BRANCH")
	if err := git.Clone(ctx.RepoURL(), cloneBranch, workDir); err != nil {
		if err := git.Clone(ctx.RepoURL(), "", workDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("failed to change to work dir: %w", err)
	}

	setupCmd := &SetupWorkspaceCmd{WorkspaceDir: "/workspace"}
	if err := setupCmd.Run(); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) syncBaseBranch(ctx *context.Context) error {
	logger.Infof("Base branch: %s", ctx.BaseBranch())

	if err := w.fetchBaseBranch(ctx); err != nil {
		logger.Warningf("failed to fetch base branch: %v", err)
		return nil
	}

	needsMerge, err := w.checkIfMergeNeeded(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if merge needed: %w", err)
	}

	if !needsMerge {
		logger.Info("Project branch is up-to-date with base branch")
		return nil
	}

	logger.Info("Project branch is behind base branch, attempting merge...")
	return w.mergeBaseBranch(ctx)
}

func (w *WorkflowCmd) fetchBaseBranch(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	logger.Infof("Fetching base branch: %s", baseBranch)

	cmd := exec.Command("git", "fetch", "origin", baseBranch+":"+baseBranch)
	if err := cmd.Run(); err != nil {
		logger.Infof("Fetch with refspec failed, falling back to plain fetch: %v", err)
		return exec.Command("git", "fetch", "origin", baseBranch).Run()
	}
	return nil
}

func (w *WorkflowCmd) checkIfMergeNeeded(ctx *context.Context) (bool, error) {
	baseBranch := ctx.BaseBranch()
	if _, err := git.RevParse("--verify", baseBranch); err != nil {
		return false, nil
	}

	mergeBase, err := git.MergeBase("HEAD", baseBranch)
	if err != nil {
		return false, err
	}

	baseCommit, err := git.RevParse(baseBranch)
	if err != nil {
		return false, err
	}

	return mergeBase != baseCommit, nil
}

func (w *WorkflowCmd) mergeBaseBranch(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	if err := git.Merge(baseBranch); err != nil {
		logger.Info("Merge had conflicts - resolving with AI...")
		_ = git.AbortMerge()

		return w.resolveConflictsWithAI(ctx)
	}

	logger.Info("Merge successful (fast-forward or no conflicts)")
	return nil
}

func (w *WorkflowCmd) resolveConflictsWithAI(ctx *context.Context) error {
	baseBranch := ctx.BaseBranch()
	logger.Info("Running AI to resolve merge conflicts...")

	instructionsFile := "./tmp/merge-instructions.md"
	instructions := fmt.Sprintf(`You need to resolve merge conflicts between the base branch (%s) and the current branch (%s).

Steps:
1. Run 'git merge %s' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. Run tests to ensure the merged code is correct
4. After resolving and verifying with tests, run 'git add <resolved-files>' and 'git commit'

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
`, baseBranch, ctx.Branch(), baseBranch)

	if err := os.MkdirAll("./tmp", 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	if err := os.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write merge instructions: %w", err)
	}

	projectPath := ctx.ProjectFile()
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join("/workspace/repo", projectPath)
	}

	ctx.SetProjectFile(projectPath)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetInstructions(instructionsFile)

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	maxIterations := resolveMaxIterations(ralphConfig, ctx.MaxIterations())
	ctx.SetMaxIterations(maxIterations)
	ctx.SetWorkflowExecution(true)

	_ = project.Execute(ctx, w.cleanupRegistrar)

	if git.HasStagedChanges() {
		logger.Info("AI did not commit the merge - committing now...")
		_ = git.StageAll()
		_ = git.Commit(fmt.Sprintf("Merge %s into %s", baseBranch, ctx.Branch()))
	}

	return nil
}

func (w *WorkflowCmd) runProject(ctx *context.Context) error {
	logger.Info("Running project...")

	projectPath := ctx.ProjectFile()
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join("/workspace/repo", projectPath)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	maxIterations := resolveMaxIterations(ralphConfig, ctx.MaxIterations())

	ctx.SetProjectFile(projectPath)
	ctx.SetMaxIterations(maxIterations)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)

	ctx.SetWorkflowExecution(true)

	if err := project.Execute(ctx, w.cleanupRegistrar); err != nil {
		return fmt.Errorf("ralph execution failed: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) displayStats() {
	logger.Info("Displaying OpenCode statistics...")
	cmd := exec.Command("opencode", "stats")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
