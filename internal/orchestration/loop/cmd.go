package loop

import (
	"github.com/zon/ralph/internal/ai"
)

// LoopConfigClient resolves the steps of the loop config entry matching the
// slug.
type LoopConfigClient interface {
	LoopSteps(slug string) ([]string, error)
}

// PromptBuilder builds the loop prompt that embeds the resolved steps.
type PromptBuilder interface {
	BuildLoopPrompt(steps []string) (string, error)
}

// SlugProposer proposes a branch slug for the given steps.
type SlugProposer interface {
	ProposeSlug(steps []string) (string, error)
}

// AIClient runs the loop prompt as one AI agent pass.
type AIClient interface {
	RunAgent(prompt string) error
}

// ReportReader reads the agent's report from report.md.
type ReportReader interface {
	ReadReport() (ai.Report, error)
}

// Cmd orchestrates the ralph loop command.
type Cmd struct {
	cfg     LoopConfigClient
	prompt  PromptBuilder
	propose SlugProposer
	ai      AIClient
	report  ReportReader
}

func NewCmd(cfg LoopConfigClient, prompt PromptBuilder, propose SlugProposer, ai AIClient, report ReportReader) *Cmd {
	return &Cmd{cfg: cfg, prompt: prompt, propose: propose, ai: ai, report: report}
}

// Result carries the resolution of a loop invocation: the branch slug and the
// steps to run.
type Result struct {
	Slug  string
	Steps []string
}

// Run resolves the branch slug and the steps to run, builds the loop prompt
// embedding the steps, and runs it as an iteration loop. The loop stops when
// the agent reports nothing to do or after max iterations, whichever comes
// first. It returns the resolution so the caller can derive the branch name
// from the slug.
func (c *Cmd) Run(slug string, steps []string, max int) (*Result, error) {
	result, err := c.resolve(slug, steps)
	if err != nil {
		return nil, err
	}
	prompt, err := c.prompt.BuildLoopPrompt(result.Steps)
	if err != nil {
		return nil, err
	}
	if err := c.iterate(prompt, max); err != nil {
		return nil, err
	}
	return result, nil
}

// iterate runs the loop prompt as an iteration loop. Each iteration invokes
// the AI and reads the agent's report. The loop stops when the report says
// nothing to do or after max iterations, whichever comes first.
func (c *Cmd) iterate(prompt string, max int) error {
	for i := 0; i < max; i++ {
		if err := c.ai.RunAgent(prompt); err != nil {
			return err
		}
		report, err := c.report.ReadReport()
		if err != nil {
			return err
		}
		if report.IsNothingToDo() {
			return nil
		}
	}
	return nil
}

// resolve returns the branch slug and steps to run. A given slug always
// consults the loop config. When steps are passed, they replace the entry's
// steps, otherwise the entry's steps are used. The given slug is returned
// unchanged. Without a slug, the slug proposer proposes one from the passed
// steps. With neither, no slug and no steps are resolved.
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

// resolveConfig returns the resolution for the given slug. Passed steps
// replace the matching loop config entry's steps when present, otherwise the
// entry's steps are used.
func (c *Cmd) resolveConfig(slug string, steps []string) (*Result, error) {
	loopSteps, err := c.cfg.LoopSteps(slug)
	if err != nil {
		return nil, err
	}
	if len(steps) > 0 {
		return &Result{Slug: slug, Steps: steps}, nil
	}
	return &Result{Slug: slug, Steps: loopSteps}, nil
}
