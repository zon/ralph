package prompt

import (
	"errors"
	"strings"

	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

// ErrMissingPrompt is returned when the workflow supplies no prompt text.
var ErrMissingPrompt = errors.New("prompt is required")

// WorkspaceSetupClient prepares the container environment before the prompt
// runs.
type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

// RunnerClient runs the prompt flow in the prepared checkout.
type RunnerClient interface {
	Run(flags WorkflowPromptFlags) error
}

// ValidatePrompt rejects a missing or whitespace-only prompt before any
// workspace work starts.
func ValidatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return ErrMissingPrompt
	}
	return nil
}

// WorkflowPromptFlags carries the workflow-side prompt invocation: the repo
// and clone branch used to prepare the workspace, the prompt branch, base
// branch, prompt text, and the model, variant, agent, service, debug, and
// verbose choices the workflow passed.
type WorkflowPromptFlags struct {
	Repo        string
	CloneBranch string
	Branch      string
	BaseBranch  string
	Prompt      string
	BotName     string
	BotEmail    string
	Model       string
	Variant     string
	Agent       string
	Debug       string
	NoServices  bool
	Verbose     bool
}

// WorkspaceFlags prepares the clone and checks out the prompt branch, creating
// it from the clone branch when it does not exist on the remote.
func (f WorkflowPromptFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:         f.Repo,
		CloneBranch:  f.CloneBranch,
		TargetBranch: f.Branch,
		CreateBranch: true,
		BotName:      f.BotName,
		BotEmail:     f.BotEmail,
	}
}

// WorkflowPromptCmd orchestrates the ralph workflow prompt command that runs
// inside the workflow container. It validates the prompt, prepares the
// workspace, and runs the prompt flow in the cloned checkout.
type WorkflowPromptCmd struct {
	workspace WorkspaceSetupClient
	runner    RunnerClient
}

func NewWorkflowPromptCmd(workspace WorkspaceSetupClient, runner RunnerClient) *WorkflowPromptCmd {
	return &WorkflowPromptCmd{workspace: workspace, runner: runner}
}

func (w *WorkflowPromptCmd) Run(flags WorkflowPromptFlags) error {
	if err := ValidatePrompt(flags.Prompt); err != nil {
		return err
	}
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	return w.runner.Run(flags)
}
