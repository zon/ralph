package cmd

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

type WorkspaceClient interface {
	ChangeDirectory(dir string) error
}

type ProjectClient interface {
	ValidateFile(path string) error
	Load(path string) (*project.Project, error)
}

type GitClient interface {
	CurrentBranch() (string, error)
	BranchName(slug string) string
}

type ConfigClient interface {
	Load() (*config.RalphConfig, error)
}

type RunnerClient interface {
	Execute(setup ExecutionSetup) error
}

type RunFlags struct {
	WorkingDir    string
	ProjectFile   string
	Follow        bool
	Local         bool
	Debug         string
	MaxIterations int
	Base          string
	Model         string
	Context       string
	Verbose       bool
	NoNotify      bool
	NoServices    bool
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

type RunCmd struct {
	workspace WorkspaceClient
	project   ProjectClient
	git       GitClient
	config    ConfigClient
	runner    RunnerClient
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
	return r.runner.Execute(setup)
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
	projectBranch := r.git.BranchName(proj.Slug)
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
