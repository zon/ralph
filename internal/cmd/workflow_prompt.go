package cmd

import (
	"os"
	"strings"

	execcontext "github.com/zon/ralph/internal/context"
	orchestrationPrompt "github.com/zon/ralph/internal/orchestration/prompt"
	"github.com/zon/ralph/internal/output"
)

// WorkflowPromptCmd is the `ralph workflow prompt` command that runs inside
// the workflow container. It validates the prompt, prepares the workspace,
// checks out the prompt branch in the cloned repository, and runs the prompt
// flow in-process.
type WorkflowPromptCmd struct {
	Repo        string `help:"GitHub repository (owner/repo)" required:""`
	CloneBranch string `help:"Branch to clone" name:"clone"`
	Branch      string `help:"Prompt branch to create or check out" name:"branch"`
	Base        string `help:"Base branch for PR creation" name:"base" short:"B"`
	Prompt      string `help:"Prompt text to run" name:"prompt"`
	BotName     string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail    string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	Debug       string `help:"Ralph branch to use for debug mode" name:"debug"`
	Model       string `help:"The model to use in format of provider/model" name:"model" short:"m"`
	Agent       string `help:"Override the opencode agent from config" name:"agent"`
	Variant     string `help:"The model variant (provider-specific reasoning effort, e.g., high, max, minimal)" name:"variant"`
	NoServices  bool   `help:"Skip service startup" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`

	// workspaceSetup prepares the container workspace before the prompt runs.
	// Tests inject a fake. When nil, Run builds the real adapter.
	workspaceSetup orchestrationPrompt.WorkspaceSetupClient `kong:"-"`

	// promptRunner runs the prompt flow in-process after the workspace is
	// prepared. Tests inject a fake. When nil, Run builds the real adapter.
	promptRunner orchestrationPrompt.RunnerClient `kong:"-"`
}

func (w *WorkflowPromptCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, w.Verbose))
	ctx.SetVerbose(w.Verbose)
	if parts := strings.SplitN(w.Repo, "/", 2); len(parts) == 2 {
		ctx.SetRepoOwner(parts[0])
		ctx.SetRepoName(parts[1])
	}
	ctx.SetBranch(w.Branch)
	ctx.SetBaseBranch(w.Base)
	ctx.SetBotName(w.BotName)
	ctx.SetBotEmail(w.BotEmail)
	ctx.SetModel(w.Model)
	ctx.SetAgent(w.Agent)
	ctx.SetVariant(w.Variant)
	ctx.SetDebugBranch(w.Debug)
	ctx.SetNoServices(w.NoServices)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	flags := orchestrationPrompt.WorkflowPromptFlags{
		Repo:        w.Repo,
		CloneBranch: w.CloneBranch,
		Branch:      w.Branch,
		BaseBranch:  w.Base,
		Prompt:      w.Prompt,
		BotName:     w.BotName,
		BotEmail:    w.BotEmail,
		Model:       w.Model,
		Variant:     w.Variant,
		Agent:       w.Agent,
		Debug:       w.Debug,
		NoServices:  w.NoServices,
		Verbose:     w.Verbose,
	}
	return w.newOrchestrationWorkflowPromptCmd(ctx).Run(flags)
}

// newOrchestrationWorkflowPromptCmd wires the container-side workflow prompt
// command. Tests inject fakes for the workspace setup and prompt runner.
func (w *WorkflowPromptCmd) newOrchestrationWorkflowPromptCmd(ctx *execcontext.Context) *orchestrationPrompt.WorkflowPromptCmd {
	workspace := w.workspaceSetup
	if workspace == nil {
		workspace = &workspaceSetupAdapter{ctx: ctx}
	}
	runner := w.promptRunner
	if runner == nil {
		runner = &promptWorkflowRunnerAdapter{ctx: ctx}
	}
	return orchestrationPrompt.NewWorkflowPromptCmd(workspace, runner)
}

// promptWorkflowRunnerAdapter runs the prompt flow in-process after the
// workspace is prepared.
type promptWorkflowRunnerAdapter struct {
	ctx *execcontext.Context
}

func (a *promptWorkflowRunnerAdapter) Run(flags orchestrationPrompt.WorkflowPromptFlags) error {
	cmd := &PromptCmd{
		ctx:        a.ctx,
		NoServices: flags.NoServices,
	}
	return cmd.RunInWorkspace(flags.Prompt, flags.Branch, flags.BaseBranch)
}
