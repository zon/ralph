package run

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type InfrastructureAdapter interface {
	RunBeforeCommands(cfg *config.RalphConfig) error
	GetCommitLog(baseBranch string, n int) (string, error)
	NotifyError(projectName string, shouldNotify bool)
	NotifySuccess(projectName string, shouldNotify bool)
	CreatePullRequest(ctx *context.Context, proj *project.Project, branchName, baseBranch, prSummary string) (string, error)
	GeneratePRSummary(ctx *context.Context, projectDesc, projectStatus, baseBranch, commitLog string) (string, error)
	LogVerbose(format string, args ...interface{})
	LogVerboseFn(func() string)
	LogSuccess(format string, args ...interface{})
}

type DefaultInfrastructureAdapter struct{}

func (a *DefaultInfrastructureAdapter) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := services.RunBefore(cfg.Before); err != nil {
			return err
		}
	}
	return nil
}

func (a *DefaultInfrastructureAdapter) GetCommitLog(baseBranch string, n int) (string, error) {
	return git.GetCommitLog(baseBranch, n)
}

func (a *DefaultInfrastructureAdapter) NotifyError(projectName string, shouldNotify bool) {
	notify.Error(projectName, shouldNotify)
}

func (a *DefaultInfrastructureAdapter) NotifySuccess(projectName string, shouldNotify bool) {
	notify.Success(projectName, shouldNotify)
}

func (a *DefaultInfrastructureAdapter) CreatePullRequest(ctx *context.Context, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	return github.CreatePullRequest(ctx, proj, branchName, baseBranch, prSummary)
}

func (a *DefaultInfrastructureAdapter) GeneratePRSummary(ctx *context.Context, projectDesc, projectStatus, baseBranch, commitLog string) (string, error) {
	return ai.GeneratePRSummary(ctx, projectDesc, projectStatus, baseBranch, commitLog)
}

func (a *DefaultInfrastructureAdapter) LogVerbose(format string, args ...interface{}) {
	logger.Verbosef(format, args...)
}

func (a *DefaultInfrastructureAdapter) LogVerboseFn(fn func() string) {
	logger.Verbose(fn())
}

func (a *DefaultInfrastructureAdapter) LogSuccess(format string, args ...interface{}) {
	logger.Successf(format, args...)
}
