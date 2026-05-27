package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

// ProjectClientAdapter wraps internal/project to satisfy the ProjectClient interface.
type ProjectClientAdapter struct{}

func NewProjectClientAdapter() *ProjectClientAdapter {
	return &ProjectClientAdapter{}
}

func (a *ProjectClientAdapter) AllRequirementsPassing(proj *project.Project) bool {
	allComplete, _, _ := project.CheckCompletion(proj)
	return allComplete
}

func (a *ProjectClientAdapter) MaxIterationsError(proj *project.Project) error {
	_, _, failingCount := project.CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
}

// AgentClientAdapter wraps internal/ai and internal/project to satisfy the AgentClient interface.
type AgentClientAdapter struct {
	ctx *execcontext.Context
}

func NewAgentClientAdapter(ctx *execcontext.Context) *AgentClientAdapter {
	return &AgentClientAdapter{ctx: ctx}
}

func (a *AgentClientAdapter) Iterate(proj *project.Project) error {
	return project.ExecuteDevelopmentIteration(a.ctx, nil)
}

func (a *AgentClientAdapter) IsFatal(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	patterns := []string{
		"Insufficient Balance",
		"insufficient balance",
		"billing",
		"account",
		"payment required",
		"quota exceeded",
	}
	for _, p := range patterns {
		if strings.Contains(errStr, p) {
			return true
		}
	}
	return false
}

func (a *AgentClientAdapter) GenerateChangelog(proj *project.Project) error {
	return ai.GenerateChangelog(a.ctx)
}

// GitClientAdapter wraps internal/git to satisfy the GitClient interface.
type GitClientAdapter struct {
	ctx         *execcontext.Context
	hasCommitted bool
}

func NewGitClientAdapter(ctx *execcontext.Context) *GitClientAdapter {
	return &GitClientAdapter{ctx: ctx}
}

func (a *GitClientAdapter) SwitchToBranch(slug string) error {
	branchName := git.SanitizeBranchName(slug)
	return git.ValidateGitStateAndSwitchBranch(a.ctx, branchName)
}

func (a *GitClientAdapter) BlockedFileExists() bool {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return false
	}
	blockedPath := filepath.Join(repoRoot, "blocked.md")
	_, err = os.Stat(blockedPath)
	return err == nil
}

func (a *GitClientAdapter) WriteBlockedFile(err error) {
	repoRoot, repoErr := git.FindRepoRoot()
	if repoErr != nil {
		return
	}
	blockedPath := filepath.Join(repoRoot, "blocked.md")
	content := fmt.Sprintf("# Blocked\n\nError: %s\n", err.Error())
	_ = os.WriteFile(blockedPath, []byte(content), 0644)
}

func (a *GitClientAdapter) HasChanges() bool {
	return git.HasUncommittedChanges()
}

func (a *GitClientAdapter) HasCommits() bool {
	return a.hasCommitted
}

func (a *GitClientAdapter) ReportExists() bool {
	_, err := os.Stat("report.md")
	return err == nil
}

func (a *GitClientAdapter) CommitFromReport(slug string) error {
	reportContent, err := os.ReadFile("report.md")
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}

	message := strings.TrimSpace(string(reportContent))

	if err := git.CommitChanges(false, "", "", message); err != nil {
		return err
	}

	if err := os.Remove("report.md"); err != nil {
		logger.Warningf("Failed to remove report.md: %v", err)
	}

	a.hasCommitted = true
	return nil
}

// GitHubClientAdapter wraps internal/github and internal/ai to satisfy the GitHubClient interface.
type GitHubClientAdapter struct {
	ctx        *execcontext.Context
	baseBranch string
}

func NewGitHubClientAdapter(ctx *execcontext.Context, baseBranch string) *GitHubClientAdapter {
	return &GitHubClientAdapter{ctx: ctx, baseBranch: baseBranch}
}

func (a *GitHubClientAdapter) CreatePR(proj *project.Project) error {
	commitLog, err := git.GetCommitLog(a.baseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	allComplete, passingCount, failingCount := project.CheckCompletion(proj)
	projectStatus := fmt.Sprintf("%d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	prSummary, err := ai.GeneratePRSummary(a.ctx, proj.Title, projectStatus, a.baseBranch, commitLog)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Slug)
	_, err = github.CreatePullRequest(a.ctx, proj, branchName, a.baseBranch, prSummary)
	if err != nil {
		if errors.Is(err, github.ErrNoCommitsBetweenBranches) {
			logger.Verbose("No commits ahead of base branch — all requirements were already passing; skipping PR creation")
			return nil
		}
		return err
	}

	return nil
}

// ServicesClientAdapter wraps internal/services to satisfy the ServicesClient interface.
type ServicesClientAdapter struct{}

func NewServicesClientAdapter() *ServicesClientAdapter {
	return &ServicesClientAdapter{}
}

func (a *ServicesClientAdapter) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		return services.RunBefore(cfg.Before)
	}
	return nil
}

// NotifyClientAdapter wraps internal/notify to satisfy the NotifyClient interface.
type NotifyClientAdapter struct {
	shouldNotify bool
}

func NewNotifyClientAdapter(shouldNotify bool) *NotifyClientAdapter {
	return &NotifyClientAdapter{shouldNotify: shouldNotify}
}

func (a *NotifyClientAdapter) Error(slug string) {
	notify.Error(slug, a.shouldNotify)
}

func (a *NotifyClientAdapter) Success(slug string) {
	notify.Success(slug, a.shouldNotify)
}

// NewRunner constructs a Runner wiring all six concrete adapters from the given context and base branch.
func NewRunner(ctx *execcontext.Context, baseBranch string) *Runner {
	return &Runner{
		project:  NewProjectClientAdapter(),
		ai:       NewAgentClientAdapter(ctx),
		git:      NewGitClientAdapter(ctx),
		github:   NewGitHubClientAdapter(ctx, baseBranch),
		services: NewServicesClientAdapter(),
		notify:   NewNotifyClientAdapter(ctx.ShouldNotify()),
	}
}
