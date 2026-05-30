package run

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
)

type AgentClient struct {
	ctx *context.Context
}

func NewAgentClient(ctx *context.Context) *AgentClient {
	return &AgentClient{ctx: ctx}
}

func (a *AgentClient) Iterate(proj *project.Project) error {
	return project.ExecuteDevelopmentIteration(a.ctx, nil)
}

func (a *AgentClient) IsFatal(err error) bool {
	return ai.IsFatalError(err)
}

func (a *AgentClient) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx)
}
