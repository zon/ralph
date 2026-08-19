package cmd

import (
	"os"
	"strings"

	orchestrationWorkflow "github.com/zon/ralph/internal/orchestration/workflowrun"
	"github.com/zon/ralph/internal/output"
)

type WorkflowRunCmd struct {
	Repo            string `help:"GitHub repository (owner/repo)" required:""`
	ProjectPath     string `help:"Path to project YAML file within the repository" required:""`
	ProjectBranch   string `help:"Branch to clone or create" name:"project-branch"`
	BaseBranch      string `help:"Base branch for PR creation" name:"base" short:"B" required:""`
	BotName         string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail        string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	Debug           string `help:"Ralph branch to use for debug mode" name:"debug"`
	Items           string `help:"jq query selecting the item array from the project file" name:"items"`
	Cleanup         bool   `help:"Delete the project file in its own commit once every item is complete" name:"cleanup"`
	NoServices      bool   `help:"Skip service startup" default:"false"`
	InstructionsMD  string `help:"Inline instructions content" name:"instructions-md"`
	ExtraIterations int    `help:"Extra iterations beyond requirement count (default: 20% of requirements)" name:"extra-iterations"`
	Model           string `help:"Override the AI model from config" name:"model"`
	Agent           string `help:"Override the opencode agent from config" name:"agent"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowRunCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	if parts := strings.SplitN(w.Repo, "/", 2); len(parts) == 2 {
		ctx.SetRepoOwner(parts[0])
		ctx.SetRepoName(parts[1])
	}
	ctx.SetBranch(w.ProjectBranch)
	ctx.SetBaseBranch(w.BaseBranch)
	ctx.SetProjectFile(w.ProjectPath)
	ctx.SetInstructionsMD(w.InstructionsMD)
	ctx.SetDebugBranch(w.Debug)
	ctx.SetBotName(w.BotName)
	ctx.SetBotEmail(w.BotEmail)
	ctx.SetModel(w.Model)
	ctx.SetAgent(w.Agent)
	ctx.SetNoServices(w.NoServices)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	cloneBranch := os.Getenv("GIT_BRANCH")

	cmd := newOrchestrationWorkflowRunCmd(ctx, w.cleanupRegistrar)
	flags := orchestrationWorkflow.WorkflowRunFlags{
		Repo:            w.Repo,
		CloneBranch:     cloneBranch,
		BaseBranch:      w.BaseBranch,
		ProjectBranch:   w.ProjectBranch,
		BotName:         w.BotName,
		BotEmail:        w.BotEmail,
		ProjectPath:     w.ProjectPath,
		InstructionsMd:  w.InstructionsMD,
		ExtraIterations: w.ExtraIterations,
		Items:           w.Items,
		Cleanup:         w.Cleanup,
		Model:           w.Model,
		Agent:           w.Agent,
		NoServices:      w.NoServices,
		Debug:           w.Debug,
	}
	return cmd.Run(flags)
}
