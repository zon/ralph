package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/requirement"
)

// RunCmd is the default command for executing ralph
type RunCmd struct {
	WorkingDir    string `help:"Working directory to run ralph in" type:"path" short:"C"`
	ProjectFile   string `arg:"" optional:"" help:"Path to project YAML file"`
	Once          bool   `help:"Single development iteration mode" default:"false"`
	MaxIterations int    `help:"Maximum number of development iterations (not applicable with --once)" default:"0"`
	DryRun        bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	NoServices    bool   `help:"Skip service startup" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
	Local         bool   `help:"Run on this machine instead of in Argo Workflows" default:"false"`
	Follow        bool   `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	Debug         string `help:"Checkout the given ralph repo branch in the workflow container and invoke ralph via 'go run' instead of the built binary" name:"debug" optional:""`
	ShowVersion   bool   `help:"Show version information" short:"v" name:"version"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the run command (implements kong.Run interface)
func (r *RunCmd) Run() error {
	// Handle version flag
	if r.ShowVersion {
		if r.date != "unknown" {
			fmt.Printf("ralph version %s (%s)\n", r.version, r.date)
		} else {
			fmt.Printf("ralph version %s\n", r.version)
		}
		return nil
	}

	// Change working directory if specified
	if r.WorkingDir != "" {
		if err := os.Chdir(r.WorkingDir); err != nil {
			return fmt.Errorf("failed to change to working directory %s: %w", r.WorkingDir, err)
		}
	}

	// Validate required fields
	if r.ProjectFile == "" {
		return fmt.Errorf("project file required (see --help)")
	}

	// Resolve the project file path relative to the (possibly changed) working directory.
	absProjectFile, err := filepath.Abs(r.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	r.ProjectFile = absProjectFile

	if _, err := os.Stat(r.ProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", r.ProjectFile)
	}

	// Validate flag combinations
	if r.Follow && r.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}

	if r.Local && r.Once {
		return fmt.Errorf("--local flag is incompatible with --once flag")
	}

	if r.Debug != "" && r.Local {
		return fmt.Errorf("--debug flag is not applicable with --local flag")
	}

	// Load config to get MaxIterations (if not explicitly provided via CLI)
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// CLI flag takes precedence over config file; if neither set, use default (10)
	maxIterations := r.MaxIterations
	if maxIterations == 0 {
		maxIterations = ralphConfig.MaxIterations
	}
	if maxIterations == 0 {
		maxIterations = 10
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile:   r.ProjectFile,
		MaxIterations: maxIterations,
		DryRun:        r.DryRun,
		Verbose:       r.Verbose,
		NoNotify:      r.NoNotify,
		NoServices:    r.NoServices,
		Local:         r.Local,
		Follow:        r.Follow,
		DebugBranch:   r.Debug,
	}

	if r.Once {
		// Execute single iteration mode
		projectName := strings.TrimSuffix(filepath.Base(ctx.ProjectFile), filepath.Ext(ctx.ProjectFile))
		if err := requirement.Execute(ctx, r.cleanupRegistrar); err != nil {
			notify.Error(projectName, ctx.ShouldNotify() && !ctx.IsDryRun())
			return err
		}

		notify.Success(projectName, ctx.ShouldNotify() && !ctx.IsDryRun())
		return nil
	}
	// Execute full orchestration mode
	return project.Execute(ctx, r.cleanupRegistrar)
}
