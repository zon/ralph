package run

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type AgentClient struct {
	ctx   *context.Context
	setup *project.IterationSetup
}

func NewAgentClient(ctx *context.Context) *AgentClient {
	return &AgentClient{ctx: ctx}
}

func (a *AgentClient) RunPicker(proj *project.Project) (string, error) {
	setup, err := project.PrepareIteration(a.ctx, nil)
	if err != nil {
		return "", err
	}
	a.setup = setup
	req, err := project.PickRequirement(a.ctx, setup)
	if err != nil {
		a.stopServices()
		return "", err
	}
	return req, nil
}

func (a *AgentClient) RunDeveloper(proj *project.Project, req string) error {
	defer a.stopServices()
	return project.DevelopRequirement(a.ctx, a.setup, req)
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

func (a *AgentClient) stopServices() {
	if a.setup != nil && a.setup.ServiceMgr != nil {
		a.setup.ServiceMgr.Stop()
	}
	a.setup = nil
}
