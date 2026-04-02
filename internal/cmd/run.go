package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
)

// RunCmd is the default command for executing ralph
type RunCmd struct {
	WorkingDir    string `help:"Working directory to run ralph in" type:"path" short:"C"`
	ProjectFile   string `arg:"" optional:"" help:"Path to project YAML file"`
	Once          bool   `help:"Single development iteration mode" default:"false"`
	MaxIterations int    `help:"Maximum number of development iterations (not applicable with --once)" default:"0"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	NoServices    bool   `help:"Skip service startup" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
	Local         bool   `help:"Run on this machine instead of in Argo Workflows" default:"false"`
	Follow        bool   `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	Debug         string `help:"Checkout the given ralph repo branch in the workflow container and invoke ralph via 'go run' instead of the built binary" name:"debug" optional:""`
	Base          string `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Model         string `help:"Override the AI model from config" name:"model" optional:""`
	Context       string `help:"Kubernetes context to use" name:"context" optional:""`
	ShowVersion   bool   `help:"Show version information" short:"v" name:"version"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the run command (implements kong.Run interface)
func (r *RunCmd) Run() error {
	if err := r.handleVersionFlag(); err != nil {
		return err
	}

	if err := r.changeWorkingDirectory(); err != nil {
		return err
	}

	if err := r.validateProjectFile(); err != nil {
		return err
	}

	if err := r.validateFlagCombinations(); err != nil {
		return err
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	maxIterations := resolveMaxIterations(ralphConfig, r.MaxIterations)

	ctx := r.createExecutionContext(maxIterations)

	// Resolve and set base branch in context
	projectData, err := project.LoadProject(r.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	projectBranch := project.SanitizeBranchName(projectData.Name)
	baseBranch := resolveBaseBranch(r.Base, currentBranch, projectBranch, ralphConfig.DefaultBranch)
	ctx.SetBaseBranch(baseBranch)

	if r.Once {
		return r.executeOnceMode(ctx)
	}
	return project.Execute(ctx, r.cleanupRegistrar)
}

func resolveBaseBranch(baseFlag, currentBranch, projectBranch, defaultBranch string) string {
	if baseFlag != "" {
		return baseFlag
	}

	if currentBranch != projectBranch {
		return currentBranch
	}

	return defaultBranch
}

func (r *RunCmd) handleVersionFlag() error {
	if !r.ShowVersion {
		return nil
	}
	if r.date != "unknown" {
		fmt.Printf("ralph version %s (%s)\n", r.version, r.date)
	} else {
		fmt.Printf("ralph version %s\n", r.version)
	}
	return nil
}

func (r *RunCmd) changeWorkingDirectory() error {
	if r.WorkingDir == "" {
		return nil
	}
	if err := os.Chdir(r.WorkingDir); err != nil {
		return fmt.Errorf("failed to change to working directory %s: %w", r.WorkingDir, err)
	}
	return nil
}

func (r *RunCmd) validateProjectFile() error {
	if r.ProjectFile == "" {
		return fmt.Errorf("project file required (see --help)")
	}

	absProjectFile, err := filepath.Abs(r.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	r.ProjectFile = absProjectFile

	if _, err := os.Stat(r.ProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", r.ProjectFile)
	}
	return nil
}

func (r *RunCmd) validateFlagCombinations() error {
	if r.Follow && r.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}

	if r.Local && r.Once {
		return fmt.Errorf("--local flag is incompatible with --once flag")
	}

	if r.Debug != "" && r.Local {
		return fmt.Errorf("--debug flag is not applicable with --local flag")
	}
	return nil
}

func (r *RunCmd) createExecutionContext(maxIterations int) *execcontext.Context {
	ctx := createExecutionContext()
	ctx.SetProjectFile(r.ProjectFile)
	ctx.SetMaxIterations(maxIterations)
	ctx.SetVerbose(r.Verbose)
	ctx.SetNoNotify(r.NoNotify)
	ctx.SetNoServices(r.NoServices)
	ctx.SetLocal(r.Local)
	ctx.SetFollow(r.Follow)
	ctx.SetDebugBranch(r.Debug)
	ctx.SetBaseBranch(r.Base)
	ctx.SetModel(r.Model)
	ctx.SetKubeContext(r.Context)
	return ctx
}

func (r *RunCmd) executeOnceMode(ctx *execcontext.Context) error {
	projectName := strings.TrimSuffix(filepath.Base(ctx.ProjectFile()), filepath.Ext(ctx.ProjectFile()))
	if err := project.ExecuteDevelopmentIteration(ctx, r.cleanupRegistrar); err != nil {
		notify.Error(projectName, ctx.ShouldNotify())
		return err
	}

	notify.Success(projectName, ctx.ShouldNotify())
	return nil
}
