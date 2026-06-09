package workspace

import "errors"

var ErrBranchNotFound = errors.New("branch not found on remote")

type GitHubClient interface {
	ConfigureAuth(repo string) error
}

type WorkspaceClient interface {
	SetupCredentials() error
	SetupSymlinks() error
}

type GitClient interface {
	ConfigureUser(name, email string)
	Clone(branch string) error
	RemoteBranchExists(branch string) (bool, error)
	FetchAndCheckout(branch string) error
	CreateAndCheckout(branch string) error
}

type WorkspaceSetup struct {
	github    GitHubClient
	workspace WorkspaceClient
	git       GitClient
}

func New(github GitHubClient, workspace WorkspaceClient, git GitClient) *WorkspaceSetup {
	return &WorkspaceSetup{
		github:    github,
		workspace: workspace,
		git:       git,
	}
}

type WorkspaceFlags struct {
	Repo         string
	CloneBranch  string
	TargetBranch string
	CreateBranch bool
	BotName      string
	BotEmail     string
	Symlinks     bool
}

func (w *WorkspaceSetup) Setup(flags WorkspaceFlags) error {
	if err := w.github.ConfigureAuth(flags.Repo); err != nil {
		return err
	}
	if err := w.workspace.SetupCredentials(); err != nil {
		return err
	}
	w.git.ConfigureUser(flags.BotName, flags.BotEmail)
	if err := w.git.Clone(flags.CloneBranch); err != nil {
		return err
	}
	if flags.TargetBranch != "" {
		if err := w.checkoutBranch(flags.TargetBranch, flags.CreateBranch); err != nil {
			return err
		}
	}
	if flags.Symlinks {
		return w.workspace.SetupSymlinks()
	}
	return nil
}

func (w *WorkspaceSetup) checkoutBranch(branch string, create bool) error {
	exists, err := w.git.RemoteBranchExists(branch)
	if err != nil {
		return err
	}
	if exists {
		return w.git.FetchAndCheckout(branch)
	}
	if !create {
		return ErrBranchNotFound
	}
	return w.git.CreateAndCheckout(branch)
}
