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

type RunnerClient interface {
	RunLocal(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error
}

type ConfigClient interface {
	LoadOptional() (*ralphcfg.RalphConfig, error)
}

type ProjectClient interface {
	Resolve(path, query string) (*ralphproj.Project, error)
}

type DebugClient interface {
	Setup(branch string) error
}

func NewWorkflowRunCmd(workspace WorkspaceSetupClient, runner RunnerClient, config ConfigClient, project ProjectClient, debug DebugClient) *WorkflowRunCmd {
	return &WorkflowRunCmd{
		workspace: workspace,
		runner:    runner,
		config:    config,
		project:   project,
		debug:     debug,
	}
}

type WorkflowRunCmd struct {
	workspace WorkspaceSetupClient
	runner    RunnerClient
	config    ConfigClient
	project   ProjectClient
	debug     DebugClient
}

type WorkflowRunFlags struct {
	Repo            string
	CloneBranch     string
	BaseBranch      string
	ProjectBranch   string
	BotName         string
	BotEmail        string
	ProjectPath     string
	InstructionsMd  string
	ExtraIterations int
	Items           string
	Model           string
	Agent           string
	NoServices      bool
	Debug           string
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
	query := cfg.ResolveItems(flags.Items)
	proj, err := w.project.Resolve(flags.ProjectPath, query)
	if err != nil {
		return err
	}
	cfg.Items = query
	w.applyFlags(proj, cfg, flags)
	return w.runner.RunLocal(proj, cfg)
}

func (w *WorkflowRunCmd) applyFlags(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig, flags WorkflowRunFlags) {
	if flags.BaseBranch != "" {
		proj.BaseBranch = flags.BaseBranch
		cfg.Base = flags.BaseBranch
	}
	if flags.InstructionsMd != "" {
		cfg.Instructions = flags.InstructionsMd
	}
	if flags.Model != "" {
		cfg.Model = flags.Model
	}
	if flags.Agent != "" {
		cfg.Agent = flags.Agent
	}
	if flags.ExtraIterations != 0 {
		v := flags.ExtraIterations
		cfg.ExtraIterations = &v
	}
	if flags.NoServices {
		cfg.Services = nil
	}
}
