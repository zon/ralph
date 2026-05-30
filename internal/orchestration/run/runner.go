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

func NewRunner(project ProjectClient, ai AgentClient, git GitClient, github GitHubClient, services ServicesClient, notify NotifyClient) *Runner {
	return &Runner{
		project:  project,
		ai:       ai,
		git:      git,
		github:   github,
		services: services,
		notify:   notify,
	}
}

func (r *Runner) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	if err := r.services.RunBeforeCommands(cfg); err != nil {
		return err
	}
	if err := r.git.SwitchToBranch(proj.Slug); err != nil {
		return err
	}
	if err := r.iterate(proj); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	if err := r.github.CreatePR(proj); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	r.notify.Success(proj.Slug)
	return nil
}

func (r *Runner) iterate(proj *project.Project) error {
	for i := 0; i < proj.MaxIterations; i++ {
		if r.project.AllRequirementsPassing(proj) {
			return nil
		}
		if r.git.BlockedFileExists() {
			return ErrBlocked
		}
		if err := r.ai.Iterate(proj); err != nil {
			if r.ai.IsFatal(err) {
				return err
			}
			r.git.WriteBlockedFile(err)
			return err
		}
		if err := r.commitIteration(proj); err != nil {
			return err
		}
	}
	return r.project.MaxIterationsError(proj)
}

func (r *Runner) commitIteration(proj *project.Project) error {
	if !r.git.HasChanges() {
		return nil
	}
	if !r.git.ReportExists() {
		if err := r.ai.GenerateChangelog(proj); err != nil {
			return err
		}
	}
	return r.git.CommitFromReport(proj.Slug)
}
