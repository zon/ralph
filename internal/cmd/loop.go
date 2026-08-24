package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/orchestration/loop"
	"github.com/zon/ralph/internal/output"
)

// LoopCmd is the `ralph loop` command. It resolves the loop config steps by
// slug or uses the passed --step values. When steps are given without a slug,
// it asks the AI for a branch slug. It builds the prompt embedding the steps
// and retains the resolved slug and steps on the command for the later loop
// phases.
type LoopCmd struct {
	Slug    string   `arg:"" optional:"" help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps   []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max     int      `help:"Maximum number of iterations" name:"max" default:"10"`
	Verbose bool     `help:"Enable verbose logging" default:"false"`

	// slugProposer proposes a branch slug from steps. Tests inject a fake. When
	// nil, Run builds the real adapter that consults the AI.
	slugProposer loop.SlugProposer `kong:"-"`

	// resolvedSlug and resolvedSteps retain the resolution of the last Run call
	// so the later loop phases (iteration, branch creation) can use them.
	resolvedSlug  string   `kong:"-"`
	resolvedSteps []string `kong:"-"`
}

// Validate checks the parsed command line before the command runs. Kong wraps
// its error in a usage error, printing the full usage alongside it.
func (c *LoopCmd) Validate() error {
	if c.Slug == "" && len(c.Steps) == 0 {
		return errors.New("a slug or at least one --step is required")
	}
	if c.Max < 1 {
		return fmt.Errorf("--max must be positive, got %d", c.Max)
	}
	return nil
}

// Run wires the orchestration, resolving the slug and steps and retaining them
// on the command so the later loop phases can use them.
func (c *LoopCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))

	propose := c.slugProposer
	if propose == nil {
		propose = &loopSlugProposer{ctx: ctx}
	}

	result, err := loop.NewCmd(&config.Client{}, &loopPromptBuilder{}, propose).Run(c.Slug, c.Steps)
	if err != nil {
		return err
	}
	c.resolvedSlug = result.Slug
	c.resolvedSteps = result.Steps
	return nil
}

// loopSlugProposer adapts ai.ProposeLoopSlug to the orchestration's
// SlugProposer interface.
type loopSlugProposer struct {
	ctx *execcontext.Context
}

// ProposeSlug asks the AI to propose a branch slug for the given steps.
func (p *loopSlugProposer) ProposeSlug(steps []string) (string, error) {
	return ai.ProposeLoopSlug(p.ctx, opencode.New(), steps)
}

// loopPromptBuilder adapts ai.BuildLoopPrompt to the orchestration's
// PromptBuilder interface.
type loopPromptBuilder struct{}

// BuildLoopPrompt builds the loop prompt embedding the given steps.
func (b *loopPromptBuilder) BuildLoopPrompt(steps []string) (string, error) {
	return ai.BuildLoopPrompt(steps)
}
