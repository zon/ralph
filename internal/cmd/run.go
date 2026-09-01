package cmd

import (
	"fmt"
	"os"

	execcontext "github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
)

// RunCmd is the default command for executing ralph
type RunCmd struct {
	WorkingDir      string `help:"Working directory to run ralph in" type:"path" short:"C"`
	InputFile       string `arg:"" optional:"" help:"Path to input file (project YAML, orchestration.md, or spec.md)"`
	ExtraIterations int    `help:"Extra iterations beyond requirement count (default: 20% of requirements)" name:"extra-iterations"`
	NoNotify        bool   `help:"Disable desktop notifications" default:"false"`
	NoServices      bool   `help:"Skip service startup" default:"false"`
	Verbose         bool   `help:"Enable verbose logging" default:"false"`
	Mode            string `help:"Execution mode: local, worktree, or remote (default: worktree)" name:"mode" optional:""`
	Follow          bool   `help:"Follow workflow logs after submission (only applicable with --mode remote)" short:"f" default:"false"`
	Debug           string `help:"Checkout the given ralph repo branch in the workflow container and invoke ralph via 'go run' instead of the built binary (only applicable with --mode remote)" name:"debug" optional:""`
	Base            string `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Items           string `help:"jq query selecting the item array from the project file (default: from config or .)" name:"items" optional:""`
	Cleanup         *bool  `help:"Delete the project file in its own commit once every item is complete" name:"cleanup"`
	Model           string `help:"Override the AI model from config" name:"model" optional:""`
	Agent           string `help:"Override the opencode agent from config" name:"agent" optional:""`
	Variant         string `help:"Override the model variant from config" name:"variant" optional:""`
	Context         string `help:"Kubernetes context to use" name:"context" optional:""`
	Namespace       string `help:"Kubernetes namespace to use" name:"namespace" short:"n" optional:""`
	ShowVersion     bool   `help:"Show version information" short:"v" name:"version"`

	version string `kong:"-"`
	date    string `kong:"-"`
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
		Mode:            r.Mode,
		Follow:          r.Follow,
		Debug:           r.Debug,
		Base:            r.Base,
		Items:           r.Items,
		Cleanup:         r.Cleanup,
		Model:           r.Model,
		Agent:           r.Agent,
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
	ctx.SetFollow(r.Follow)
	ctx.SetDebugBranch(r.Debug)
	ctx.SetBaseBranch(r.Base)
	ctx.SetModel(r.Model)
	ctx.SetAgent(r.Agent)
	ctx.SetVariant(r.Variant)
	ctx.SetKubeContext(r.Context)
	ctx.SetKubeNamespace(r.Namespace)
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
