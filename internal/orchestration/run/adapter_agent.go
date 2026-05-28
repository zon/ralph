package run

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
)

// AgentClientAdapter adapts project and ai functions to the AgentClient interface.
type AgentClientAdapter struct {
	ctx *context.Context
}

func NewAgentClientAdapter(ctx *context.Context) *AgentClientAdapter {
	return &AgentClientAdapter{ctx: ctx}
}

func (a *AgentClientAdapter) Iterate(proj *project.Project) error {
	return project.ExecuteDevelopmentIteration(a.ctx, nil)
}

func (a *AgentClientAdapter) IsFatal(err error) bool {
	return ai.IsFatalError(err)
}

func (a *AgentClientAdapter) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx)
}
