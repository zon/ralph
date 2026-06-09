package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type ProjectClient interface {
	Reload(proj *project.Project) *project.Project
	AllRequirementsPassing(proj *project.Project) bool
	MaxIterationsError(proj *project.Project) error
	HasChanges(proj *project.Project) bool
	NormalizeAndStage(proj *project.Project)
	HasSpec(proj *project.Project) bool
	HasOrchestration(proj *project.Project) bool
	RemoveOrchestration(proj *project.Project) error
}

type AIClient interface {
	RunPicker(proj *project.Project) (string, error)
	RunDeveloper(proj *project.Project, req string) error
	IsFatal(err error) bool
	GenerateChangelog(proj *project.Project) error
	FixServiceStartup(cfg *config.RalphConfig, err error) error
	PrintStats()
	WriteOrchestration(input *project.InputFile) error
	WriteProject(input *project.InputFile) (*project.Project, error)
}

type EnvClient interface {
	InWorkflow() bool
}

type GitClient interface {
	SwitchToBranch(slug string) error
	BlockedFileExists() bool
	WriteBlockedFile(err error)
	HasChanges() bool
	ReportExists() bool
	CommitFromReport(slug string) error
	CurrentBranch() (string, error)
	IsBranchSyncedWithRemote(branch string) error
	CommitOrchestrationRemoval(slug string) error
	CommitGeneratedArtifacts(slug string) error
}

type WorkflowClient interface {
	Submit(input *project.InputFile, cloneBranch string, debug string) (string, error)
	FollowLogs(workflowName string) error
	PrintLogHint(workflowName string)
}

type GitHubClient interface {
	CreatePR(proj *project.Project) error
}

type ServicesClient interface {
	RunBeforeCommands(cfg *config.RalphConfig) error
	Start(cfg *config.RalphConfig) (*services.Manager, error)
	Stop(svc *services.Manager)
	RemoveLogs(cfg *config.RalphConfig)
}

type NotifyClient interface {
	Error(slug string)
	Success(slug string)
}

type Runner struct {
	project  ProjectClient
	ai       AIClient
	git      GitClient
	github   GitHubClient
	services ServicesClient
	notify   NotifyClient
	env      EnvClient
}

func NewRunner(project ProjectClient, ai AIClient, git GitClient, github GitHubClient, services ServicesClient, notify NotifyClient, env EnvClient) *Runner {
	return &Runner{
		project:  project,
		ai:       ai,
		git:      git,
		github:   github,
		services: services,
		notify:   notify,
		env:      env,
	}
}

func (r *Runner) Env() EnvClient {
	return r.env
}

func (r *Runner) RunLocal(input *project.InputFile, cfg *config.RalphConfig) error {
	if r.env.InWorkflow() {
		defer r.ai.PrintStats()
	}
	if err := r.services.RunBeforeCommands(cfg); err != nil {
		return err
	}
	if err := r.git.SwitchToBranch(input.Slug()); err != nil {
		return err
	}
	proj, err := r.generateArtifacts(input)
	if err != nil {
		r.notify.Error(input.Slug())
		return err
	}
	if err := r.iterate(proj, cfg); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	if err := r.removeOrchestration(proj); err != nil {
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

func (r *Runner) generateArtifacts(input *project.InputFile) (*project.Project, error) {
	if input.IsProject() {
		return input.Project(), nil
	}
	if input.IsSpec() {
		if err := r.ai.WriteOrchestration(input); err != nil {
			return nil, err
		}
	}
	proj, err := r.ai.WriteProject(input)
	if err != nil {
		return nil, err
	}
	return proj, r.git.CommitGeneratedArtifacts(proj.Slug)
}

func (r *Runner) iterate(proj *project.Project, cfg *config.RalphConfig) error {
	for i := 0; i < proj.MaxIterations; i++ {
		proj = r.project.Reload(proj)
		if r.project.AllRequirementsPassing(proj) {
			return nil
		}
		if r.git.BlockedFileExists() {
			return ErrBlocked
		}
		if err := r.runIteration(proj, cfg); err != nil {
			return err
		}
		if err := r.commitIteration(proj); err != nil {
			return err
		}
	}
	proj = r.project.Reload(proj)
	if r.project.AllRequirementsPassing(proj) {
		return nil
	}
	return r.project.MaxIterationsError(proj)
}

func (r *Runner) runIteration(proj *project.Project, cfg *config.RalphConfig) error {
	if sv, ok := r.ai.(interface{ setLastVariant(string) }); ok {
		sv.setLastVariant(cfg.Variant)
	}
	svc, err := r.services.Start(cfg)
	if err != nil {
		if fixErr := r.ai.FixServiceStartup(cfg, err); fixErr != nil {
			return fixErr
		}
		svc = nil
	}
	defer r.services.Stop(svc)
	defer r.services.RemoveLogs(cfg)
	req, err := r.ai.RunPicker(proj)
	if err != nil {
		return r.blockAndReturn(err)
	}
	if err := r.ai.RunDeveloper(proj, req); err != nil {
		return r.blockAndReturn(err)
	}
	return r.cleanup(proj)
}

func (r *Runner) cleanup(proj *project.Project) error {
	if r.project.HasChanges(proj) {
		r.project.NormalizeAndStage(proj)
	}
	return nil
}

func (r *Runner) blockAndReturn(err error) error {
	if !r.ai.IsFatal(err) {
		r.git.WriteBlockedFile(err)
	}
	return err
}

func (r *Runner) removeOrchestration(proj *project.Project) error {
	if !r.project.HasSpec(proj) {
		return nil
	}
	if !r.project.HasOrchestration(proj) {
		return nil
	}
	if err := r.project.RemoveOrchestration(proj); err != nil {
		return err
	}
	return r.git.CommitOrchestrationRemoval(proj.Slug)
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
