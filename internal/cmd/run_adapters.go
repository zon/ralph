package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/run"
	"github.com/zon/ralph/internal/workspace"
)

type workspaceAdapter struct{}

func (a *workspaceAdapter) ChangeDirectory(path string) error {
	if path == "" {
		return nil
	}
	return workspace.Chdir(path)
}

type configAdapter struct{}

func (a *configAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type projectAdapter struct{}

func (a *projectAdapter) Load(path string) (*project.Project, error) {
	return project.LoadProject(path)
}

func (a *projectAdapter) ValidateFile(path string) error {
	if path == "" {
		return fmt.Errorf("project file required (see --help)")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absPath)
	}
	return nil
}

type localRunnerAdapter struct {
	ctx *execcontext.Context
}

func (a *localRunnerAdapter) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	runner := run.NewLocalRunner(a.ctx, proj.BaseBranch)
	return runner.RunLocal(proj, cfg)
}

type remoteRunnerAdapter struct {
	ctx *execcontext.Context
}

func (a *remoteRunnerAdapter) RunRemote(proj *project.Project, follow bool) error {
	runner := run.NewRemoteRunner(a.ctx)
	return runner.RunRemote(proj, follow)
}

func newOrchestrationRunCmd(ctx *execcontext.Context) *orchestrationRun.RunCmd {
	return orchestrationRun.NewRunCmd(
		&workspaceAdapter{},
		&projectAdapter{},
		git.NewClient(ctx),
		&configAdapter{},
		&localRunnerAdapter{ctx: ctx},
		&remoteRunnerAdapter{ctx: ctx},
	)
}
