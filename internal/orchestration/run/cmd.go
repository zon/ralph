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
	RunLocal(input *project.InputFile, cfg *config.RalphConfig) error
}

type RemoteRunnerClient interface {
	Run(input *project.InputFile, flags RunRemoteFlags) error
}

type ExecutionSetup struct {
	Config        *config.RalphConfig
	Mode          string
	BranchName    string
	CurrentBranch string
	BaseBranch    string
	Model         string
	Agent         string
	Context       string
}

type RunFlags struct {
	WorkingDir      string
	InputFile       string
	ExtraIterations int
	Items           string
	Cleanup         *bool
	Mode            string
	Follow          bool
	Debug           string
	Base            string
	Model           string
	Agent           string
	Context         string
}

// Validate rejects the workflow-only --follow and --debug flags for the local
// and worktree execution modes.
func (f RunFlags) Validate(mode string) error {
	if mode == config.ModeRemote {
		return nil
	}
	if f.Follow {
		return fmt.Errorf("--follow flag is not applicable with --mode %s", mode)
	}
	if f.Debug != "" {
		return fmt.Errorf("--debug flag is not applicable with --mode %s", mode)
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
	setup, err := r.prepareSetup(flags, input)
	if err != nil {
		return err
	}
	if err := flags.Validate(setup.Mode); err != nil {
		return err
	}
	if setup.Mode == config.ModeRemote {
		return r.remote.Run(input, RunRemoteFlags{
			Follow:     flags.Follow,
			Debug:      flags.Debug,
			BaseBranch: setup.BaseBranch,
			Items:      setup.Config.Items,
			Cleanup:    setup.Config.Cleanup,
		})
	}
	return r.local.RunLocal(input, setup.Config)
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
	cfg.Items = cfg.ResolveItems(flags.Items)
	cfg.Cleanup = cfg.ResolveCleanup(flags.Cleanup)
	cfg.Base = baseBranch
	mode, err := cfg.ResolveMode(flags.Mode)
	if err != nil {
		return ExecutionSetup{}, err
	}
	cfg.Mode = mode
	return ExecutionSetup{
		Config:        cfg,
		Mode:          mode,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
		Model:         flags.Model,
		Agent:         flags.Agent,
		Context:       flags.Context,
	}, nil
}
