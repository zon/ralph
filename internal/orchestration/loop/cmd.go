package loop

import "github.com/zon/ralph/internal/config"

// PromptBuilder builds the loop prompt that embeds the resolved steps.
type PromptBuilder interface {
	BuildLoopPrompt(steps []string) (string, error)
}

// SlugProposer proposes a branch slug for the given steps.
type SlugProposer interface {
	ProposeSlug(steps []string) (string, error)
}

// Cmd orchestrates the ralph loop command.
type Cmd struct {
	cfg     config.Loader
	prompt  PromptBuilder
	propose SlugProposer
}

func NewCmd(cfg config.Loader, prompt PromptBuilder, propose SlugProposer) *Cmd {
	return &Cmd{cfg: cfg, prompt: prompt, propose: propose}
}

// Result carries the resolution of a loop invocation: the branch slug and the
// steps to run.
type Result struct {
	Slug  string
	Steps []string
}

// Run resolves the branch slug and the steps to run, builds the loop prompt
// embedding the steps, and returns the resolution so the caller can derive the
// branch name from the slug.
func (c *Cmd) Run(slug string, steps []string) (*Result, error) {
	result, err := c.resolve(slug, steps)
	if err != nil {
		return nil, err
	}
	if _, err := c.prompt.BuildLoopPrompt(result.Steps); err != nil {
		return nil, err
	}
	return result, nil
}

// resolve returns the branch slug and steps to run. A given slug always
// consults the loop config, whose entry's steps are used unless steps are
// passed, which replace them. The given slug is returned unchanged. Without a
// slug, the slug proposer proposes one from the passed steps. With neither, no
// slug and no steps are resolved.
func (c *Cmd) resolve(slug string, steps []string) (*Result, error) {
	if slug != "" {
		return c.resolveConfig(slug, steps)
	}
	if len(steps) == 0 {
		return &Result{}, nil
	}
	proposed, err := c.propose.ProposeSlug(steps)
	if err != nil {
		return nil, err
	}
	return &Result{Slug: proposed, Steps: steps}, nil
}

// resolveConfig returns the resolution for the given slug: the passed steps
// replace the matching loop config entry's steps when present, otherwise the
// entry's steps are used.
func (c *Cmd) resolveConfig(slug string, steps []string) (*Result, error) {
	cfg, err := c.cfg.Load()
	if err != nil {
		return nil, err
	}
	resolved, err := cfg.LoopSteps(slug)
	if err != nil {
		return nil, err
	}
	if len(steps) > 0 {
		return &Result{Slug: slug, Steps: steps}, nil
	}
	return &Result{Slug: slug, Steps: resolved}, nil
}
