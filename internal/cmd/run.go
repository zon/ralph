package cmd

import (
	"fmt"
	"os"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
)

// RunCmd is the default command for executing ralph
type RunCmd struct {
	WorkingDir       string `help:"Working directory to run ralph in" type:"path" short:"C"`
	InputFile        string `arg:"" optional:"" help:"Path to input file (project YAML, orchestration.md, or spec.md)"`
	ExtraIterations  int    `help:"Extra iterations beyond requirement count (default: 20% of requirements)" name:"extra-iterations"`
	NoNotify         bool   `help:"Disable desktop notifications" default:"false"`
	NoServices       bool   `help:"Skip service startup" default:"false"`
	Verbose          bool   `help:"Enable verbose logging" default:"false"`
	Local            bool   `help:"Run on this machine instead of in Argo Workflows" default:"false"`
	Follow           bool   `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	Debug            string `help:"Checkout the given ralph repo branch in the workflow container and invoke ralph via 'go run' instead of the built binary" name:"debug" optional:""`
	Base             string `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Model            string `help:"Override the AI model from config" name:"model" optional:""`
	Variant          string `help:"Override the model variant from config" name:"variant" optional:""`
	Context          string `help:"Kubernetes context to use" name:"context" optional:""`
	ShowVersion      bool   `help:"Show version information" short:"v" name:"version"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the run command (implements kong.Run interface)
func (r *RunCmd) Run() error {
	if err := r.handleVersionFlag(); err != nil {
		return err
	}

	ctx := r.newExecutionContext()

	flags := orchestrationRun.RunFlags{
		WorkingDir:      r.WorkingDir,
		InputFile:       r.InputFile,
		ExtraIterations: r.ExtraIterations,
		Local:           r.Local,
		Follow:          r.Follow,
		Debug:           r.Debug,
		Base:            r.Base,
		Model:           r.Model,
		Context:         r.Context,
	}

	cmd := newOrchestrationRunCmd(ctx)
	return cmd.Run(flags)
}

func (r *RunCmd) newExecutionContext() *execcontext.Context {
	ctx := createExecutionContext()
	ctx.SetProjectFile(r.InputFile)
	ctx.SetVerbose(r.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, r.Verbose))
	ctx.SetNoNotify(r.NoNotify)
	ctx.SetNoServices(r.NoServices)
	ctx.SetLocal(r.Local)
	ctx.SetFollow(r.Follow)
	ctx.SetDebugBranch(r.Debug)
	ctx.SetBaseBranch(r.Base)
	ctx.SetModel(r.Model)
	ctx.SetVariant(r.Variant)
	ctx.SetKubeContext(r.Context)
	return ctx
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
