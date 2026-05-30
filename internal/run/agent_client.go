package run

import (
	"fmt"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type AgentClient struct {
	ctx *context.Context
}

func NewAgentClient(ctx *context.Context) *AgentClient {
	return &AgentClient{ctx: ctx}
}

func (a *AgentClient) RunPicker(proj *project.Project) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	pickedReqPath := filepath.Join(filepath.Dir(proj.Path), "picked-requirement.yaml")

	setup := &project.IterationSetup{
		Project:       proj,
		CommitLog:     commitLog,
		PickedReqPath: pickedReqPath,
	}

	return project.PickRequirement(a.ctx, setup)
}

func (a *AgentClient) RunDeveloper(proj *project.Project, req string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	commitLog, err := getCommitLog(a.ctx, cfg.DefaultBranch)
	if err != nil {
		commitLog = ""
	}

	setup := &project.IterationSetup{
		Project:   proj,
		CommitLog: commitLog,
		Config:    cfg,
	}

	return project.DevelopRequirement(a.ctx, setup, req)
}

func (a *AgentClient) IsFatal(err error) bool {
	return ai.IsFatalError(err)
}

func (a *AgentClient) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx)
}

func (a *AgentClient) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	svcMgr := services.NewManager()
	if failedSvc, startErr := svcMgr.Start(cfg.Services); startErr != nil {
		fixPrompt, buildErr := ai.BuildFixServicePrompt(a.ctx, failedSvc, startErr)
		if buildErr != nil {
			return buildErr
		}
		return ai.RunAgent(a.ctx, fixPrompt)
	}
	return nil
}

func getCommitLog(ctx *context.Context, defaultBranch string) (string, error) {
	baseBranch := defaultBranch
	if ctx.BaseBranch() != "" {
		baseBranch = ctx.BaseBranch()
	}
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", err
	}
	if currentBranch == baseBranch {
		return "", nil
	}
	return git.GetCommitLog(baseBranch, 10)
}
