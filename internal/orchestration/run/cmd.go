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
	config    config.Loader
	local     LocalRunnerClient
	remote    RemoteRunnerClient
}

type WorkspaceClient interface {
	ChangeDirectory(path string) error
}

type ProjectRepo interface {
	ResolveInputFile(path string) (*project.InputFile, error)
}

type LocalRunnerClient interface {
	RunLocal(input *project.InputFile, cfg *config.RalphConfig, baseBranch string) error
}

type RemoteRunnerClient interface {
	Run(input *project.InputFile, flags RunRemoteFlags) error
}

type ExecutionSetup struct {
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
	Model         string
	Context       string
}

type RunFlags struct {
	WorkingDir      string
	InputFile       string
	ExtraIterations int
	Local           bool
	Follow          bool
	Debug           string
	Base            string
	Model           string
	Context         string
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

func NewRunCmd(workspace WorkspaceClient, project ProjectRepo, git GitClient, config config.Loader, local LocalRunnerClient, remote RemoteRunnerClient) *RunCmd {
	return &RunCmd{
		workspace: workspace,
		project:   project,
		git:       git,
		config:    config,
		local:     local,
		remote:    remote,
	}
}

func (r *RunCmd) Run(flags RunFlags) error {
	if err := r.workspace.ChangeDirectory(flags.WorkingDir); err != nil {
		return err
	}
	input, err := r.project.ResolveInputFile(flags.InputFile)
	if err != nil {
		return err
	}
	if err := flags.Validate(); err != nil {
		return err
	}
	setup, err := r.prepareSetup(flags, input)
	if err != nil {
		return err
	}
	if flags.Local {
		return r.local.RunLocal(input, setup.Config, setup.BaseBranch)
	}
	return r.remote.Run(input, RunRemoteFlags{Follow: flags.Follow, Debug: flags.Debug, BaseBranch: setup.BaseBranch})
}

func (r *RunCmd) prepareSetup(flags RunFlags, input *project.InputFile) (ExecutionSetup, error) {
	cfg, err := r.config.Load()
	if err != nil {
		return ExecutionSetup{}, err
	}
	currentBranch, err := r.git.CurrentBranch()
	if err != nil {
		return ExecutionSetup{}, err
	}
	projectBranch := git.SanitizeBranchName(input.Slug())
	baseBranch := resolveBaseBranch(flags.Base, currentBranch, projectBranch, cfg.DefaultBranch)
	if flags.ExtraIterations != 0 {
		v := flags.ExtraIterations
		cfg.ExtraIterations = &v
	}
	return ExecutionSetup{
		Config:        cfg,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
		Model:         flags.Model,
		Context:       flags.Context,
	}, nil
}
