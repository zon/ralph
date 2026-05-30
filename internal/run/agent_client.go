package run

import (
	"fmt"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type AgentClient struct {
	ctx       *context.Context
	collector *opencode.SessionCollector
}

func NewAgentClient(ctx *context.Context) *AgentClient {
	collector := &opencode.SessionCollector{}
	goCtx := opencode.WithSessionCollector(ctx.GoContext(), collector)
	ctx = ctx.WithGoContext(goCtx)
	return &AgentClient{ctx: ctx, collector: collector}
}

func (a *AgentClient) PrintStats() {
	ids := a.collector.IDs()
	if len(ids) == 0 {
		return
	}
	var totalInput, totalOutput int64
	var totalCost float64
	for _, id := range ids {
		stats, err := opencode.ExportSession(id)
		if err != nil {
			logger.Warningf("failed to export session %s: %v", id, err)
			continue
		}
		totalInput += stats.InputTokens
		totalOutput += stats.OutputTokens
		totalCost += stats.Cost
	}
	logger.Infof("Session stats: %d sessions, %d input tokens, %d output tokens, $%.6f total cost", len(ids), totalInput, totalOutput, totalCost)
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
