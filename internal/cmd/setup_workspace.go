package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/orchestration/setup_workspace"
	"github.com/zon/ralph/internal/output"
)

type SetupWorkspaceCmd struct {
	WorkspaceDir string `help:"Workspace directory containing mounted config files" default:"/workspace"`
	out          *output.Client
}

func (s *SetupWorkspaceCmd) Run() error {
	if s.out == nil {
		s.out = output.NewClient(os.Stdout, os.Stderr, false)
	}
	orchestrator := newSetupWorkspaceOrchestrator(s.WorkspaceDir, s.out)
	return orchestrator.Run()
}

type setupWorkspaceConfigLoaderAdapter struct{}

func (a *setupWorkspaceConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type setupWorkspaceFsClientAdapter struct{}

func (a *setupWorkspaceFsClientAdapter) Getwd() (string, error) {
	return os.Getwd()
}

func (a *setupWorkspaceFsClientAdapter) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (a *setupWorkspaceFsClientAdapter) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (a *setupWorkspaceFsClientAdapter) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (a *setupWorkspaceFsClientAdapter) Symlink(oldName, newName string) error {
	return os.Symlink(oldName, newName)
}

type setupWorkspaceLoggerAdapter struct {
	out *output.Client
}

func (a *setupWorkspaceLoggerAdapter) Infof(format string, args ...interface{}) {
	a.out.Infof(format, args...)
}

func newSetupWorkspaceOrchestrator(workspaceDir string, out *output.Client) *setup_workspace.SetupWorkspaceCmd {
	return setup_workspace.New(
		workspaceDir,
		&setupWorkspaceConfigLoaderAdapter{},
		&setupWorkspaceFsClientAdapter{},
		&setupWorkspaceLoggerAdapter{out: out},
	)
}
