package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

type ProjectClient interface {
	Resolve(path string, query string) (*project.Project, error)
	Complete(proj *project.Project, base string) ([]int, error)
	Incomplete(proj *project.Project, base string) ([]project.Item, error)
	ExtraIterations(proj *project.Project, cfg *config.RalphConfig) int
	IncompleteError(proj *project.Project, base string) error
	Remove(proj *project.Project) error
	HasSpec(proj *project.Project) bool
	HasOrchestration(proj *project.Project) bool
	RemoveOrchestration(proj *project.Project) error
}

type AIClient interface {
	RunPicker(proj *project.Project, incomplete []project.Item) (project.Item, error)
	RunDeveloper(proj *project.Project, item project.Item) error
	IsFatal(err error) bool
	GenerateChangelog(proj *project.Project) error
	FixServiceStartup(cfg *config.RalphConfig, err error) error
	PrintStats()
	WriteOrchestration(input *project.InputFile) error
	WriteProject(input *project.InputFile) (string, error)
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
	CommitProjectRemoval(path string) error
}

type WorkflowClient interface {
	Submit(input *project.InputFile, cloneBranch string, debug string, baseBranch string, items string, cleanup bool) (string, error)
	FollowLogs(workflowName string) error
	PrintLogHint(workflowName string)
}

type GitHubClient interface {
	CreatePR(proj *project.Project, head string) error
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

func (r *Runner) Project() ProjectClient {
	return r.project
}

// RunLocal runs the full development loop in the current checkout, switching
// to the project branch first.
func (r *Runner) RunLocal(input *project.InputFile, cfg *config.RalphConfig) error {
	return r.runLocal(input, cfg, false)
}

// RunLocalInWorktree runs the full development loop inside an existing worktree
// that already has the project branch checked out. The branch switch is skipped
// so the run neither changes the current checkout nor checks a freshly created
// worktree branch against the remote.
func (r *Runner) RunLocalInWorktree(input *project.InputFile, cfg *config.RalphConfig) error {
	return r.runLocal(input, cfg, true)
}

func (r *Runner) runLocal(input *project.InputFile, cfg *config.RalphConfig, inWorktree bool) error {
	if r.env.InWorkflow() {
		defer r.ai.PrintStats()
	}
	if err := r.services.RunBeforeCommands(cfg); err != nil {
		return err
	}
	if !inWorktree {
		if err := r.git.SwitchToBranch(input.Slug()); err != nil {
			return err
		}
	}
	proj, err := r.generateArtifacts(input, cfg)
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
	if err := r.removeProjectFile(proj, cfg); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	if err := r.github.CreatePR(proj, git.SanitizeBranchName(proj.Slug)); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	r.notify.Success(proj.Slug)
	return nil
}

func (r *Runner) generateArtifacts(input *project.InputFile, cfg *config.RalphConfig) (*project.Project, error) {
	if input.IsProject() {
		return r.project.Resolve(input.Path(), cfg.Items)
	}
	if input.IsSpec() {
		if err := r.ai.WriteOrchestration(input); err != nil {
			return nil, err
		}
	}
	path, err := r.ai.WriteProject(input)
	if err != nil {
		return nil, err
	}
	proj, err := r.project.Resolve(path, cfg.Items)
	if err != nil {
		return nil, err
	}
	return proj, r.git.CommitGeneratedArtifacts(proj.Slug)
}

func (r *Runner) iterate(proj *project.Project, cfg *config.RalphConfig) error {
	extra := r.project.ExtraIterations(proj, cfg)
	limit := len(proj.Items) + extra
	for i := 0; i < limit; i++ {
		incomplete, err := r.project.Incomplete(proj, cfg.Base)
		if err != nil {
			return err
		}
		if len(incomplete) == 0 {
			return nil
		}
		if r.git.BlockedFileExists() {
			return ErrBlocked
		}
		if err := r.runIteration(proj, incomplete, cfg); err != nil {
			return err
		}
		if err := r.commitIteration(proj); err != nil {
			return err
		}
	}
	return r.project.IncompleteError(proj, cfg.Base)
}

func (r *Runner) runIteration(proj *project.Project, incomplete []project.Item, cfg *config.RalphConfig) error {
	svc, err := r.services.Start(cfg)
	if err != nil {
		if fixErr := r.ai.FixServiceStartup(cfg, err); fixErr != nil {
			return fixErr
		}
		svc = nil
	}
	defer r.services.Stop(svc)
	defer r.services.RemoveLogs(cfg)
	item, err := r.ai.RunPicker(proj, incomplete)
	if err != nil {
		return r.blockAndReturn(err)
	}
	if err := r.ai.RunDeveloper(proj, item); err != nil {
		return r.blockAndReturn(err)
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

func (r *Runner) removeProjectFile(proj *project.Project, cfg *config.RalphConfig) error {
	if !cfg.Cleanup {
		return nil
	}
	if err := r.project.Remove(proj); err != nil {
		return err
	}
	return r.git.CommitProjectRemoval(proj.Path)
}

func (r *Runner) commitIteration(proj *project.Project) error {
	if r.git.ReportExists() {
		return r.git.CommitFromReport(proj.Slug)
	}
	if !r.git.HasChanges() {
		return nil
	}
	if err := r.ai.GenerateChangelog(proj); err != nil {
		return err
	}
	return r.git.CommitFromReport(proj.Slug)
}
