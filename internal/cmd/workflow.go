package cmd

import (
	"os"

	orchestrationWorkflow "github.com/zon/ralph/internal/orchestration/workflowrun"
)

type WorkflowRunCmd struct {
	Repo           string `help:"GitHub repository (owner/repo)" required:""`
	ProjectPath    string `help:"Path to project YAML file within the repository" required:""`
	ProjectBranch  string `help:"Branch to clone or create" name:"project-branch"`
	BaseBranch     string `help:"Base branch for PR creation" name:"base" short:"B" required:""`
	BotName        string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail       string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	Debug          string `help:"Ralph branch to use for debug mode" name:"debug"`
	NoServices     bool   `help:"Skip service startup" default:"false"`
	InstructionsMD string `help:"Inline instructions content" name:"instructions-md"`
	MaxIterations  int    `help:"Maximum number of iterations" name:"max-iterations" default:"0"`
	Model          string `help:"Override the AI model from config" name:"model"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowRunCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetRepo(w.Repo)
	ctx.SetBranch(w.ProjectBranch)
	ctx.SetBaseBranch(w.BaseBranch)
	ctx.SetProjectFile(w.ProjectPath)
	ctx.SetInstructionsMD(w.InstructionsMD)
	ctx.SetDebugBranch(w.Debug)
	ctx.SetMaxIterations(w.MaxIterations)
	ctx.SetBotName(w.BotName)
	ctx.SetBotEmail(w.BotEmail)
	ctx.SetModel(w.Model)
	ctx.SetNoServices(w.NoServices)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	cloneBranch := os.Getenv("GIT_BRANCH")

	cmd := newOrchestrationWorkflowRunCmd(ctx, w.cleanupRegistrar)
	flags := orchestrationWorkflow.WorkflowRunFlags{
		Repo:           w.Repo,
		CloneBranch:    cloneBranch,
		BaseBranch:     w.BaseBranch,
		ProjectBranch:  w.ProjectBranch,
		BotName:        w.BotName,
		BotEmail:       w.BotEmail,
		ProjectPath:    w.ProjectPath,
		InstructionsMd: w.InstructionsMD,
		MaxIterations:  w.MaxIterations,
		Model:          w.Model,
		NoServices:     w.NoServices,
		Debug:          w.Debug,
	}
	return cmd.Run(flags)
}
