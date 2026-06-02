package cmd

import (
	gocontext "context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/orchestration/workflow"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
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

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowCmd) Run() error {
	if w.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}

	ctx := createExecutionContext()
	ctx.SetVerbose(w.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, w.Verbose))
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

	orchestrator := newWorkflowOrchestrator(ctx)
	return orchestrator.Run(ctx, w.cleanupRegistrar)
}

type gitHubAuthAdapter struct{}

func (a *gitHubAuthAdapter) ConfigureGitAuth(ctx gocontext.Context, owner, repo, secretsDir string) error {
	return github.ConfigureGitAuth(ctx, owner, repo, secretsDir)
}

type openCodeAdapter struct{}

func (a *openCodeAdapter) SetupOpenCodeCredentials(out *output.Client) error {
	return workspace.SetupOpenCodeCredentials(out)
}

type gitClientAdapter struct{}

func (a *gitClientAdapter) Config(global bool, key, value string) error {
	return git.Config(global, key, value)
}

func (a *gitClientAdapter) FetchBranch(out *output.Client, branch string) error {
	return git.FetchBranch(out, branch)
}

func (a *gitClientAdapter) RevParse(args ...string) (string, error) {
	return git.RevParse(args...)
}

func (a *gitClientAdapter) MergeBase(x, y string) (string, error) {
	return git.MergeBase(x, y)
}

func (a *gitClientAdapter) Merge(branch string) error {
	return git.Merge(branch)
}

func (a *gitClientAdapter) AbortMerge() error {
	return git.AbortMerge()
}

func (a *gitClientAdapter) TmpPath(name string) (string, error) {
	return git.TmpPath(name)
}

func (a *gitClientAdapter) GetCurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

func (a *gitClientAdapter) SanitizeBranchName(name string) string {
	return git.SanitizeBranchName(name)
}

type workspaceAdapter struct{}

func (a *workspaceAdapter) PrepareWorkspace(out *output.Client, repoURL, branch, workDir string) error {
	return workspace.PrepareWorkspace(out, repoURL, branch, workDir)
}

type workspaceSetupAdapter struct {
	out         *output.Client
	workspaceDir string
}

func (a *workspaceSetupAdapter) Run() error {
	cmd := &SetupWorkspaceCmd{WorkspaceDir: a.workspaceDir, out: a.out}
	return cmd.Run()
}

type configLoaderAdapter struct{}

func (a *configLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type projectLoaderAdapter struct{}

func (a *projectLoaderAdapter) Load(path string) (*project.Project, error) {
	return project.LoadProject(path)
}

type executorAdapter struct{}

func (a *executorAdapter) Execute(ctx *context.Context, cleanupRegistrar func(func()), setup *workflow.ProjectExecutionSetup) error {
	cmdSetup := &ExecutionSetup{
		ProjectFile:   setup.ProjectFile,
		Project:       setup.Project,
		Config:        setup.Config,
		BranchName:    setup.BranchName,
		CurrentBranch: setup.CurrentBranch,
		BaseBranch:    setup.BaseBranch,
	}
	return Execute(ctx, cleanupRegistrar, cmdSetup)
}

func newWorkflowOrchestrator(ctx *context.Context) *workflow.Workflow {
	return workflow.New(
		&gitHubAuthAdapter{},
		&openCodeAdapter{},
		&gitClientAdapter{},
		&workspaceAdapter{},
		&workspaceSetupAdapter{out: ctx.Output(), workspaceDir: workspace.DefaultWorkspaceDir},
		&configLoaderAdapter{},
		&projectLoaderAdapter{},
		&executorAdapter{},
	)
}
