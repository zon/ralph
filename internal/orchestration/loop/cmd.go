package loop

import "github.com/zon/ralph/internal/config"

// PromptBuilder builds the loop prompt that embeds the resolved steps.
type PromptBuilder interface {
	BuildLoopPrompt(steps []string) (string, error)
}

// Cmd orchestrates the ralph loop command.
type Cmd struct {
	cfg    config.Loader
	prompt PromptBuilder
}

func NewCmd(cfg config.Loader, prompt PromptBuilder) *Cmd {
	return &Cmd{cfg: cfg, prompt: prompt}
}

// Run resolves the steps to run and builds the loop prompt embedding them.
func (c *Cmd) Run(slug string, steps []string) error {
	resolved, err := c.resolveSteps(slug, steps)
	if err != nil {
		return err
	}
	_, err = c.prompt.BuildLoopPrompt(resolved)
	return err
}

// resolveSteps returns the steps of the loop config entry matching the slug,
// or the passed steps when the slug is empty.
func (c *Cmd) resolveSteps(slug string, steps []string) ([]string, error) {
	if slug == "" {
		return steps, nil
	}
	cfg, err := c.cfg.Load()
	if err != nil {
		return nil, err
	}
	return cfg.LoopSteps(slug)
}
