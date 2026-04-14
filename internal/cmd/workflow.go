package cmd

import (
	gocontext "context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/run"
	"github.com/zon/ralph/internal/workspace"
)

const (
	DefaultSecretsDir = "/secrets/github"
)

type WorkflowCmd struct {
	Repo           string `arg:"" help:"GitHub repository (owner/repo)" required:""`
	ProjectPath    string `arg:"" help:"Path to project YAML file within the repository" optional:""`
	ProjectBranch  string `help:"Branch to clone or create" name:"project-branch" optional:""`
	BaseBranch     string `help:"Base branch for PR creation" name:"base" short:"B" required:""`
	BotName        string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail       string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	DebugBranch    string `help:"Ralph branch to use for debug mode (clones ralph from this branch and runs via go run)" name:"debug" optional:""`
	Verbose        bool   `help:"Enable verbose logging" default:"false"`
	NoServices     bool   `help:"Skip service startup" default:"false"`
	InstructionsMD string `help:"Inline instructions content" name:"instructions-md" optional:""`
	MaxIterations  int    `help:"Maximum number of iterations" name:"max-iterations" default:"0"`
	Model          string `help:"Override the AI model from config" name:"model" optional:""`
	Review         bool   `help:"Run review mode instead of a project" name:"review" default:"false"`
	Filter         string `help:"Only run review items whose text, file, or url property contains this string" name:"filter" optional:""`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowCmd) Run() error {
	if !w.Review && w.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}

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

	if err := workspace.SetupOpenCodeCredentials(); err != nil {
		return fmt.Errorf("failed to setup OpenCode credentials: %w", err)
	}

	w.configureGitUser(ctx)

	if err := w.cloneAndSetupRepo(ctx); err != nil {
		return fmt.Errorf("failed to clone and setup repo: %w", err)
	}

	if w.Review {
		if err := w.runReview(ctx); err != nil {
			return fmt.Errorf("failed to run review: %w", err)
		}
	} else {
		if err := w.syncBaseBranch(ctx); err != nil {
			return fmt.Errorf("failed to sync base branch: %w", err)
		}

		if err := w.runProject(ctx); err != nil {
			return fmt.Errorf("failed to run project: %w", err)
		}
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

func (w *WorkflowCmd) configureGitUser(ctx *context.Context) {
	logger.Info("Configuring git user...")
	_ = git.Config(true, "user.name", ctx.BotName())
	_ = git.Config(true, "user.email", ctx.BotEmail())
}

func (w *WorkflowCmd) cloneAndSetupRepo(ctx *context.Context) error {
	cloneBranch := os.Getenv("GIT_BRANCH")
	if err := workspace.PrepareWorkspace(ctx.RepoURL(), cloneBranch, workspace.DefaultWorkDir); err != nil {
		return err
	}

	setupCmd := &SetupWorkspaceCmd{WorkspaceDir: workspace.DefaultWorkspaceDir}
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
	return git.FetchBranch(baseBranch)
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

	instructions := fmt.Sprintf(`You need to resolve merge conflicts between the base branch (%s) and the current branch (%s).

Steps:
1. Run 'git merge %s' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. Run tests to ensure the merged code is correct
4. After resolving and verifying with tests, run 'git add <resolved-files>' to stage them (the system will automatically commit)

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
`, baseBranch, ctx.Branch(), baseBranch)

	instructionsFile, err := git.TmpPath("merge-instructions.md")
	if err != nil {
		return fmt.Errorf("failed to get tmp path for merge instructions: %w", err)
	}

	if err := os.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write merge instructions: %w", err)
	}

	ctx.SetBaseBranch(baseBranch)

	return w.prepareAndExecute(ctx, w.cleanupRegistrar, instructionsFile)
}

func (w *WorkflowCmd) runProject(ctx *context.Context) error {
	logger.Info("Running project...")

	return w.prepareAndExecute(ctx, w.cleanupRegistrar, "")
}

func (w *WorkflowCmd) prepareAndExecute(ctx *context.Context, cleanupRegistrar func(func()), instructionsFile string) error {
	projectPath := ctx.ProjectFile()
	if !filepath.IsAbs(projectPath) {
		var err error
		projectPath, err = filepath.Abs(projectPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project file path: %w", err)
		}
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

	if instructionsFile != "" {
		ctx.SetInstructions(instructionsFile)
	}

	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	projectBranch := git.SanitizeBranchName(proj.Name)
	setup := &run.ExecutionSetup{
		ProjectFile:   projectPath,
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    ctx.BaseBranch(),
	}

	if err := run.Execute(ctx, cleanupRegistrar, setup); err != nil {
		return fmt.Errorf("ralph execution failed: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) runReview(ctx *context.Context) error {
	logger.Info("Running review...")
	reviewCmd := &ReviewRunCmd{
		Local:   true,
		Verbose: w.Verbose,
		Model:   w.Model,
		Base:    ctx.BaseBranch(),
		Filter:  w.Filter,
	}
	return reviewCmd.Run()
}

func (w *WorkflowCmd) displayStats() {
	logger.Info("Displaying OpenCode statistics...")
	ai.DisplayStats()
}
