package comment

import (
	"errors"

	ralphcfg "github.com/zon/ralph/internal/config"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	"github.com/zon/ralph/internal/services"
)

var ErrMissingCommentBody = errors.New("comment body is required")

type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

type ConfigClient interface {
	LoadOptional() (*ralphcfg.RalphConfig, error)
}

type AIClient interface {
	RenderCommentPrompt(ctx CommentContext, instructionsFile string) (string, error)
	RunAgent(prompt string) error
	GenerateChangelog() error
	GenerateCommentReply(ctx CommentContext, pushed bool) (string, error)
}

type ServicesClient interface {
	Start(cfg *ralphcfg.RalphConfig) (*services.Manager, error)
	Stop(svc *services.Manager)
}

type GitClient interface {
	HasChanges() bool
	ReportExists() bool
	CommitAndPushFromReport() error
}

type GitHubClient interface {
	PostComment(prNumber int, body string) error
}

type CommentContext struct {
	CommentBody string
	PRNumber    int
	PRBranch    string
	RepoOwner   string
	RepoName    string
}

func NewWorkflowCommentCmd(workspace WorkspaceSetupClient, config ConfigClient, ai AIClient, services ServicesClient, git GitClient, github GitHubClient) *WorkflowCommentCmd {
	return &WorkflowCommentCmd{
		workspace: workspace,
		config:    config,
		ai:        ai,
		services:  services,
		git:       git,
		github:    github,
	}
}

type WorkflowCommentCmd struct {
	workspace WorkspaceSetupClient
	config    ConfigClient
	ai        AIClient
	services  ServicesClient
	git       GitClient
	github    GitHubClient
}

type WorkflowCommentFlags struct {
	Repo             string
	CloneBranch      string
	ProjectBranch    string
	BotName          string
	BotEmail         string
	CommentBody      string
	PRNumber         int
	RepoOwner        string
	RepoName         string
	NoServices       bool
	InstructionsFile string
}

func (f WorkflowCommentFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:        f.Repo,
		CloneBranch: f.CloneBranch,
		BotName:     f.BotName,
		BotEmail:    f.BotEmail,
	}
}

func (f WorkflowCommentFlags) CommentContext() CommentContext {
	return CommentContext{
		CommentBody: f.CommentBody,
		PRNumber:    f.PRNumber,
		PRBranch:    f.ProjectBranch,
		RepoOwner:   f.RepoOwner,
		RepoName:    f.RepoName,
	}
}

func (w *WorkflowCommentCmd) Run(flags WorkflowCommentFlags) error {
	if flags.CommentBody == "" {
		return ErrMissingCommentBody
	}
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	cfg, err := w.config.LoadOptional()
	if err != nil {
		return err
	}
	if flags.NoServices {
		cfg.Services = nil
	}
	prompt, err := w.ai.RenderCommentPrompt(flags.CommentContext(), flags.InstructionsFile)
	if err != nil {
		return err
	}
	if !flags.NoServices {
		svc, err := w.services.Start(cfg)
		if err != nil {
			return err
		}
		defer w.services.Stop(svc)
	}
	if err := w.ai.RunAgent(prompt); err != nil {
		return err
	}
	pushed, err := w.commitChanges()
	if err != nil {
		return err
	}
	reply, err := w.ai.GenerateCommentReply(flags.CommentContext(), pushed)
	if err != nil {
		return err
	}
	return w.github.PostComment(flags.PRNumber, reply)
}

func (w *WorkflowCommentCmd) commitChanges() (bool, error) {
	if !w.git.HasChanges() {
		return false, nil
	}
	if !w.git.ReportExists() {
		if err := w.ai.GenerateChangelog(); err != nil {
			return false, err
		}
	}
	if err := w.git.CommitAndPushFromReport(); err != nil {
		return false, err
	}
	return true, nil
}
