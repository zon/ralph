package run

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

type RunCmd struct {
	workspace WorkspaceClient
	project   ProjectRepo
	git       GitClient
	config    ConfigClient
	local     LocalRunnerClient
	remote    RemoteRunnerClient
}

type WorkspaceClient interface {
	ChangeDirectory(path string) error
}

type ConfigClient interface {
	Load() (*config.RalphConfig, error)
}

type ProjectRepo interface {
	Load(path string) (*project.Project, error)
	ValidateFile(path string) error
}

type LocalRunnerClient interface {
	RunLocal(proj *project.Project, cfg *config.RalphConfig) error
}

type RemoteRunnerClient interface {
	RunRemote(proj *project.Project, follow bool) error
}

type ExecutionSetup struct {
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
	MaxIterations int
	Model         string
	Context       string
}

type RunFlags struct {
	WorkingDir    string
	ProjectFile   string
	MaxIterations int
	Local         bool
	Follow        bool
	Debug         string
	Base          string
	Model         string
	Context       string
}

func (f RunFlags) Validate() error {
	if f.Follow && f.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}
	if f.Debug != "" && f.Local {
		return fmt.Errorf("--debug flag is not applicable with --local flag")
	}
	return nil
}

func (r *RunCmd) Run(flags RunFlags) error {
	if err := r.workspace.ChangeDirectory(flags.WorkingDir); err != nil {
		return err
	}
	if err := r.project.ValidateFile(flags.ProjectFile); err != nil {
		return err
	}
	if err := flags.Validate(); err != nil {
		return err
	}
	setup, err := r.prepareSetup(flags)
	if err != nil {
		return err
	}
	setup.Project.MaxIterations = setup.MaxIterations
	setup.Project.BaseBranch = setup.BaseBranch
	if flags.Local {
		return r.local.RunLocal(setup.Project, setup.Config)
	}
	return r.remote.RunRemote(setup.Project, flags.Follow)
}

func (r *RunCmd) prepareSetup(flags RunFlags) (ExecutionSetup, error) {
	cfg, err := r.config.Load()
	if err != nil {
		return ExecutionSetup{}, err
	}
	proj, err := r.project.Load(flags.ProjectFile)
	if err != nil {
		return ExecutionSetup{}, err
	}
	currentBranch, err := r.git.CurrentBranch()
	if err != nil {
		return ExecutionSetup{}, err
	}
	projectBranch := git.BranchName(proj.Slug)
	return ExecutionSetup{
		Project:       proj,
		Config:        cfg,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    resolveBaseBranch(flags.Base, currentBranch, projectBranch, cfg.DefaultBranch),
		MaxIterations: resolveMaxIterations(cfg.MaxIterations, flags.MaxIterations),
		Model:         flags.Model,
		Context:       flags.Context,
	}, nil
}
