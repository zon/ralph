package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

type ProjectClient interface {
	AllRequirementsPassing(proj *project.Project) bool
	MaxIterationsError(proj *project.Project) error
}

type AgentClient interface {
	Iterate(proj *project.Project) error
	IsFatal(err error) bool
	GenerateChangelog(proj *project.Project) error
}

type GitClient interface {
	SwitchToBranch(slug string) error
	BlockedFileExists() bool
	WriteBlockedFile(err error)
	HasChanges() bool
	ReportExists() bool
	CommitFromReport(slug string) error
}

type GitHubClient interface {
	CreatePR(proj *project.Project) error
}

type ServicesClient interface {
	RunBeforeCommands(cfg *config.RalphConfig) error
}

type NotifyClient interface {
	Error(slug string)
	Success(slug string)
}

type Runner struct {
	project  ProjectClient
	ai       AgentClient
	git      GitClient
	github   GitHubClient
	services ServicesClient
	notify   NotifyClient
}
