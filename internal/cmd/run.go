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
	WorkingDir      string `help:"Working directory to run Ralph in" type:"path" short:"C"`
	InputFile       string `arg:"" optional:"" help:"Path to input file (project YAML, orchestration.md, or spec.md)"`
	ExtraIterations int    `help:"Extra iterations beyond project item count (default: 20% item count)" name:"extra"`
	NoNotify        bool   `help:"Disable desktop notifications" default:"false"`
	NoServices      bool   `help:"Skip service startup" default:"false"`
	Verbose         bool   `help:"Enable verbose logging" default:"false"`
	Mode            string `help:"Execution mode: local, worktree, or remote (default: worktree)" name:"mode" optional:""`
	Follow          bool   `help:"Follow workflow logs after submission (only applicable with --mode remote)" short:"f" default:"false"`
	Debug           string `help:"Checkout the given Ralph repo branch in the workflow container and invoke Ralph via 'go run' instead of the built binary (only applicable with --mode remote)" name:"debug" optional:""`
	Base            string `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Items           string `help:"jq query selecting the project item list (default: .)" name:"items" optional:"" short:"i"`
	Cleanup         *bool  `help:"Delete the project file in its own commit once every item is complete" name:"cleanup"`
	Model           string `help:"The model to use in format of provider/model" name:"model" optional:"" short:"m"`
	Agent           string `help:"Override the opencode agent from config" name:"agent" optional:""`
	Variant         string `help:"The model variant (provider-specific reasoning effort, e.g., high, max, minimal)" name:"variant" optional:""`
	Context         string `help:"The name of the Kubernetes context to use" name:"context" optional:""`
	Namespace       string `help:"The name of the Kubernetes namespace to use" name:"namespace" short:"n" optional:""`
	ShowVersion     bool   `help:"Show version information" short:"v" name:"version"`

	version string `kong:"-"`
	date    string `kong:"-"`
}

// Run executes the run command (implements kong.Run interface)
func (r *RunCmd) Run() error {
	if r.ShowVersion {
		r.printVersion()
		return nil
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

func (r *RunCmd) printVersion() {
	if r.date != "unknown" {
		fmt.Printf("%s (%s)\n", r.version, r.date)
	} else {
		fmt.Printf("%s\n", r.version)
	}
}
