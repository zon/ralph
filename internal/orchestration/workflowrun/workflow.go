package workflowrun

import (
	"errors"

	ralphcfg "github.com/zon/ralph/internal/config"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	ralphproj "github.com/zon/ralph/internal/project"
)

var ErrMissingProjectPath = errors.New("project path is required")

type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

type GitClient interface {
	FetchBranch(branch string) error
	NeedsMerge(branch string) (bool, error)
	Merge(branch string) error
	AbortMerge()
}

type AIClient interface {
	ResolveMergeConflicts(baseBranch, projectBranch string) error
}

type RunnerClient interface {
	RunLocal(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error
}

type ConfigClient interface {
	LoadOptional() (*ralphcfg.RalphConfig, error)
}

type ProjectClient interface {
	Load(path string) (*ralphproj.Project, error)
}

type DebugClient interface {
	Setup(branch string) error
}

func NewWorkflowRunCmd(workspace WorkspaceSetupClient, git GitClient, ai AIClient, runner RunnerClient, config ConfigClient, project ProjectClient, debug DebugClient) *WorkflowRunCmd {
	return &WorkflowRunCmd{
		workspace: workspace,
		git:       git,
		ai:        ai,
		runner:    runner,
		config:    config,
		project:   project,
		debug:     debug,
	}
}

type WorkflowRunCmd struct {
	workspace WorkspaceSetupClient
	git       GitClient
	ai        AIClient
	runner    RunnerClient
	config    ConfigClient
	project   ProjectClient
	debug     DebugClient
}

type WorkflowRunFlags struct {
	Repo           string
	CloneBranch    string
	BaseBranch     string
	ProjectBranch  string
	BotName        string
	BotEmail       string
	ProjectPath    string
	InstructionsMd string
	MaxIterations  int
	Model          string
	NoServices     bool
	Debug          string
}

func (f WorkflowRunFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:        f.Repo,
		CloneBranch: f.CloneBranch,
		BotName:     f.BotName,
		BotEmail:    f.BotEmail,
	}
}

func (w *WorkflowRunCmd) Run(flags WorkflowRunFlags) error {
	if flags.ProjectPath == "" {
		return ErrMissingProjectPath
	}
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	if flags.Debug != "" {
		if err := w.debug.Setup(flags.Debug); err != nil {
			return err
		}
	}
	cfg, err := w.config.LoadOptional()
	if err != nil {
		return err
	}
	proj, err := w.project.Load(flags.ProjectPath)
	if err != nil {
		return err
	}
	w.applyFlags(proj, cfg, flags)
	if err := w.syncBaseBranch(flags.BaseBranch, flags.ProjectBranch); err != nil {
		return err
	}
	return w.runner.RunLocal(proj, cfg)
}

func (w *WorkflowRunCmd) syncBaseBranch(baseBranch, projectBranch string) error {
	if err := w.git.FetchBranch(baseBranch); err != nil {
		return nil
	}
	needsMerge, err := w.git.NeedsMerge(baseBranch)
	if err != nil {
		return err
	}
	if !needsMerge {
		return nil
	}
	if err := w.git.Merge(baseBranch); err != nil {
		w.git.AbortMerge()
		return w.ai.ResolveMergeConflicts(baseBranch, projectBranch)
	}
	return nil
}

func (w *WorkflowRunCmd) applyFlags(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig, flags WorkflowRunFlags) {
	if flags.BaseBranch != "" {
		proj.BaseBranch = flags.BaseBranch
	}
	if flags.MaxIterations > 0 {
		proj.MaxIterations = flags.MaxIterations
	}
	if flags.InstructionsMd != "" {
		cfg.Instructions = flags.InstructionsMd
	}
	if flags.Model != "" {
		cfg.Model = flags.Model
	}
	if flags.NoServices {
		cfg.Services = nil
	}
}


