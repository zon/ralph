package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

// Version information
var (
	Version = "0.1.0"
	Date    = "unknown"
)

// CLI defines the command-line interface structure
type CLI struct {
	Run     RunCmd     `cmd:"" help:"Full orchestration: create branch, iterate development, submit PR"`
	Once    OnceCmd    `cmd:"" help:"Single development iteration"`
	Version VersionCmd `cmd:"" help:"Show version information"`
}

// RunCmd represents the 'ralph run' command
type RunCmd struct {
	ProjectFile   string `arg:"" help:"Path to project YAML file" type:"path"`
	MaxIterations int    `help:"Maximum number of development iterations" default:"10"`
	DryRun        bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
}

// Run executes the run command
func (r *RunCmd) Run(ctx *kong.Context) error {
	fmt.Printf("Running full orchestration for: %s\n", r.ProjectFile)
	fmt.Printf("Max iterations: %d\n", r.MaxIterations)
	if r.DryRun {
		fmt.Println("[DRY-RUN] Simulating execution...")
	}
	// TODO: Implement full orchestration logic
	return fmt.Errorf("not implemented yet")
}

// OnceCmd represents the 'ralph once' command
type OnceCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file" type:"path"`
	DryRun      bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify    bool   `help:"Disable desktop notifications" default:"false"`
	NoServices  bool   `help:"Skip service startup" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
}

// Run executes the once command
func (o *OnceCmd) Run(ctx *kong.Context) error {
	fmt.Printf("Running single iteration for: %s\n", o.ProjectFile)
	if o.DryRun {
		fmt.Println("[DRY-RUN] Simulating execution...")
	}
	// TODO: Implement single iteration logic
	return fmt.Errorf("not implemented yet")
}

// VersionCmd represents the 'ralph version' command
type VersionCmd struct{}

// Run executes the version command
func (v *VersionCmd) Run(ctx *kong.Context) error {
	if Date != "unknown" {
		fmt.Printf("ralph version %s (%s)\n", Version, Date)
	} else {
		fmt.Printf("ralph version %s\n", Version)
	}
	return nil
}

func main() {
	cli := &CLI{}

	ctx := kong.Parse(cli,
		kong.Name("ralph"),
		kong.Description("AI-powered development orchestration tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Execute the selected command
	err := ctx.Run(ctx)
	ctx.FatalIfErrorf(err)
}
