package command

import (
	"errors"

	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

var ErrMissingCommand = errors.New("command cannot be empty")

type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

type ExecClient interface {
	Run(tokens []string) error
}

type WorkflowCommandCmd struct {
	workspace WorkspaceSetupClient
	exec      ExecClient
}

type WorkflowCommandFlags struct {
	Repo        string
	CloneBranch string
	BotName     string
	BotEmail    string
	Command     []string
}

func (f WorkflowCommandFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:        f.Repo,
		CloneBranch: f.CloneBranch,
		BotName:     f.BotName,
		BotEmail:    f.BotEmail,
	}
}

func (w *WorkflowCommandCmd) Run(flags WorkflowCommandFlags) error {
	if len(flags.Command) == 0 {
		return ErrMissingCommand
	}
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	return w.exec.Run(flags.Command)
}
