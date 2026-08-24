package cmd

import (
	"errors"
	"fmt"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/orchestration/loop"
)

// LoopCmd is the `ralph loop` command. It resolves the loop config steps by
// slug (or uses the passed --step values) and builds the prompt embedding them.
type LoopCmd struct {
	Slug  string   `arg:"" optional:"" help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max   int      `help:"Maximum number of iterations" name:"max" default:"10"`
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

// Run wires the orchestration. The iteration loop that executes the prompt is
// a later work item.
func (c *LoopCmd) Run() error {
	return loop.NewCmd(&config.Client{}, &LoopPromptBuilder{}).Run(c.Slug, c.Steps)
}

// LoopPromptBuilder adapts ai.BuildLoopPrompt to the orchestration's
// PromptBuilder interface.
type LoopPromptBuilder struct{}

// BuildLoopPrompt builds the loop prompt embedding the given steps.
func (b *LoopPromptBuilder) BuildLoopPrompt(steps []string) (string, error) {
	return ai.BuildLoopPrompt(steps)
}
