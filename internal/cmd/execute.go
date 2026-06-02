package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

type ExecutionSetup struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}

type CommandSetup struct {
	Command []string
	Config  *config.RalphConfig
}

func PrepareExecution(ctx *context.Context) (*ExecutionSetup, error) {
	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project file path: %w", err)
	}

	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load project file: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Slug)
	ctx.Output().Debugf("Branch name: %s", branchName)

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	baseBranch := ctx.BaseBranch()
	if baseBranch == "" {
		baseBranch = ralphConfig.DefaultBranch
	}

	return &ExecutionSetup{
		ProjectFile:   absProjectFile,
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    branchName,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
	}, nil
}

func ExecuteCommand(ctx *context.Context, cleanupRegistrar func(func()), setup *CommandSetup) error {
	if !ctx.IsLocal() {
		return executeCommandRemote(ctx, setup)
	}

	if err := infrastructureRunBeforeCommands(ctx.Output(), setup.Config); err != nil {
		return err
	}

	if err := runCommand(setup.Command); err != nil {
		notify.Error("command", ctx.ShouldNotify())
		return err
	}

	notify.Success("command", ctx.ShouldNotify())
	return nil
}

func executeCommandRemote(ctx *context.Context, setup *CommandSetup) error {
	ctx.Output().Debug("Submitting Argo Workflow for command...")

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	ctx.Output().Debugf("Checking branch '%s' is in sync with remote...", currentBranch)
	if err := git.IsBranchSyncedWithRemote(currentBranch); err != nil {
		return err
	}

	ctx.Output().Debug("Generating command workflow...")
	var remoteURL string
	if ctx.Repo() != "" {
		owner, name := ctx.RepoOwnerAndName()
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		remoteURL, err = git.RemoteURL()
		if err != nil {
			return fmt.Errorf("failed to get remote URL: %w", err)
		}
	}

	wf, err := workflow.GenerateCommandWorkflow(ctx, currentBranch, remoteURL)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	if ctx.IsVerbose() {
		workflowYAML, _ := wf.Render()
		ctx.Output().Debugf("Generated workflow YAML:\n%s", workflowYAML)
	}

	argoClient := argo.NewClient()
	workflowName, err := wf.Submit(ctx.GoContext(), argoClient)
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %w", err)
	}

	ctx.Output().Successf("Workflow submitted: %s", workflowName)

	if ctx.ShouldFollow() {
		if err := argoClient.FollowLogs(argo.K8sContext{Name: wf.KubeContext, Namespace: wf.Namespace}, workflowName); err != nil {
			notify.Error("command", ctx.ShouldNotify())
			return fmt.Errorf("argo logs failed: %w", err)
		}
		notify.Success("command", ctx.ShouldNotify())
	} else {
		ctx.Output().Infof("To follow logs, run: argo logs -n %s %s -f", wf.Namespace, workflowName)
	}

	return nil
}

func Execute(ctx *context.Context, cleanupRegistrar func(func()), setup *ExecutionSetup) error {
	if !ctx.IsLocal() {
		return NewRemoteRunner(ctx).RunRemote(setup.Project, ctx.ShouldFollow())
	}

	return NewLocalRunner(ctx, setup.BaseBranch).RunLocal(setup.Project, setup.Config)
}

func infrastructureRunBeforeCommands(out *output.Client, cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := services.RunBefore(out, cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}

func runCommand(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("command required")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}


