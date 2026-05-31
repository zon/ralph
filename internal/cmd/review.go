package cmd

import (
	orchestrationReview "github.com/zon/ralph/internal/orchestration/review"
)

type ReviewRunCmd struct {
	Model   string `help:"Override the AI model from config" name:"model" optional:""`
	Base    string `help:"Override the base branch for PR creation" name:"base" optional:"" short:"B"`
	Local   bool   `help:"Run on this machine instead of submitting to Argo Workflows" default:"false"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Context string `help:"Kubernetes context to use" name:"context" optional:""`
	Seed    int64  `help:"Random seed for shuffling review items (0 = random)" default:"0"`
	Follow  bool   `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	Filter  string `help:"Only run review items whose text, file, or url property contains this string" name:"filter" optional:""`
	One     bool   `help:"Randomly pick one review item and run only that one" name:"one" default:"false"`
}

type ReviewCmd struct {
	Run          ReviewRunCmd    `cmd:"" default:"withargs" help:"Run AI-powered code reviews from config prompts"`
	Architecture ArchitectureCmd `cmd:"" help:"Generate an architecture.yaml file summarizing the repo structure"`
}

func (r *ReviewRunCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(r.Model)
	ctx.SetLocal(r.Local)
	ctx.SetFollow(r.Follow)
	ctx.SetKubeContext(r.Context)
	ctx.SetFilter(r.Filter)

	flags := orchestrationReview.ReviewFlags{
		Seed:    r.Seed,
		Filter:  r.Filter,
		One:     r.One,
		Base:    r.Base,
		Local:   r.Local,
		Follow:  r.Follow,
		Model:   r.Model,
		Verbose: r.Verbose,
	}

	cmd := newOrchestrationReviewCmd(ctx)
	return cmd.Run(flags)
}
