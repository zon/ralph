package cmd

import (
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/architecture"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

type ArchitectureCmd struct {
	Output  string `help:"Output path for architecture.yaml" default:"architecture.yaml"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Model   string `help:"Override the AI model from config" name:"model" optional:""`
}

func (r *ArchitectureCmd) Run() error {
	if r.Verbose {
		logger.SetVerbose(true)
	}

	ctx := createExecutionContext()
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(r.Model)

	prompt, err := ai.BuildArchitecturePrompt(r.Output)
	if err != nil {
		return fmt.Errorf("failed to build architecture prompt: %w", err)
	}

	if err := ai.RunAgent(ctx, prompt); err != nil {
		return fmt.Errorf("architecture generation failed: %w", err)
	}

	if err := r.validateAndFix(ctx); err != nil {
		return err
	}

	logger.Successf("Architecture written to %s", r.Output)

	if err := r.commitAndCreatePR(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ArchitectureCmd) validateAndFix(ctx *context.Context) error {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		arch, err := architecture.Load(r.Output)
		if err != nil {
			if attempt < maxAttempts {
				fixPrompt, promptErr := ai.BuildArchitectureFixPrompt(r.Output, []string{err.Error()})
				if promptErr != nil {
					return fmt.Errorf("failed to build fix prompt: %w", promptErr)
				}
				if fixErr := ai.RunAgent(ctx, fixPrompt); fixErr != nil {
					return fmt.Errorf("architecture fix attempt %d failed: %w", attempt, fixErr)
				}
				continue
			}
			return fmt.Errorf("failed to load architecture file after %d attempts: %w", maxAttempts, err)
		}

		validationErrors := arch.Validate()
		if len(validationErrors) == 0 {
			return nil
		}

		if attempt < maxAttempts {
			fixPrompt, promptErr := ai.BuildArchitectureFixPrompt(r.Output, validationErrors)
			if promptErr != nil {
				return fmt.Errorf("failed to build fix prompt: %w", promptErr)
			}
			if fixErr := ai.RunAgent(ctx, fixPrompt); fixErr != nil {
				return fmt.Errorf("architecture fix attempt %d failed: %w", attempt, fixErr)
			}
			continue
		}

		return fmt.Errorf("architecture validation failed after %d attempts: %s", maxAttempts, strings.Join(validationErrors, "; "))
	}

	return nil
}

func (r *ArchitectureCmd) commitAndCreatePR(ctx *context.Context) error {
	if !ctx.IsWorkflowExecution() {
		return nil
	}

	if !git.IsFileModifiedOrNew(r.Output) {
		logger.Infof("No changes detected for %s, skipping PR", r.Output)
		return nil
	}

	if err := git.CheckoutOrCreateBranch("architecture"); err != nil {
		return fmt.Errorf("failed to checkout architecture branch: %w", err)
	}

	if err := git.StageFile(r.Output); err != nil {
		return fmt.Errorf("failed to stage architecture file: %w", err)
	}

	owner, repo := ctx.RepoOwnerAndName()
	auth := &git.AuthConfig{Owner: owner, Repo: repo}

	if err := git.CommitAllAndPush(auth, "architecture", "architecture: generate architecture.yaml"); err != nil {
		return fmt.Errorf("failed to commit and push architecture: %w", err)
	}

	baseBranch := ctx.BaseBranch()
	if baseBranch == "" {
		baseBranch = "main"
	}

	proj := &project.Project{
		Slug:  "architecture",
		Title: "Update architecture.yaml",
	}

	prSummary := "Automatically generated architecture.yaml documenting the project structure."

	prURL, err := github.CreatePullRequest(ctx, proj, "architecture", baseBranch, prSummary)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Successf("Pull request created: %s", prURL)
	return nil
}
