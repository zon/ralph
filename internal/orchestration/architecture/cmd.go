package architecture

import (
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/architecture"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

type AIClient interface {
	BuildArchitecturePrompt(output string) (string, error)
	BuildArchitectureFixPrompt(output string, errors []string) (string, error)
	RunAgent(prompt string) error
}

type GitClient interface {
	IsFileModifiedOrNew(path string) bool
	CheckoutOrCreateBranch(name string) error
	StageFile(path string) error
	CommitAllAndPush(auth *git.AuthConfig, branchName, commitMsg string) error
}

type GitHubClient interface {
	CreatePullRequest(proj *project.Project, branchName, baseBranch, prSummary string) (string, error)
}

type ArchitectureClient interface {
	Load(path string) (*architecture.Architecture, error)
}

type ArchitectureFlags struct {
	Output              string
	Verbose             bool
	Model               string
	IsWorkflowExecution bool
	RepoOwner           string
	RepoName            string
	BaseBranch          string
}

type ArchitectureCmd struct {
	ai         AIClient
	git        GitClient
	github     GitHubClient
	archClient ArchitectureClient
}

func NewArchitectureCmd(ai AIClient, git GitClient, github GitHubClient, archClient ArchitectureClient) *ArchitectureCmd {
	return &ArchitectureCmd{
		ai:         ai,
		git:        git,
		github:     github,
		archClient: archClient,
	}
}

func (r *ArchitectureCmd) Run(flags ArchitectureFlags) error {
	if flags.Verbose {
		logger.SetVerbose(true)
	}

	prompt, err := r.ai.BuildArchitecturePrompt(flags.Output)
	if err != nil {
		return fmt.Errorf("failed to build architecture prompt: %w", err)
	}

	if err := r.ai.RunAgent(prompt); err != nil {
		return fmt.Errorf("architecture generation failed: %w", err)
	}

	if err := r.validateAndFix(flags); err != nil {
		return err
	}

	logger.Successf("Architecture written to %s", flags.Output)

	if err := r.commitAndCreatePR(flags); err != nil {
		return err
	}

	return nil
}

func (r *ArchitectureCmd) validateAndFix(flags ArchitectureFlags) error {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		arch, err := r.archClient.Load(flags.Output)
		if err != nil {
			if attempt < maxAttempts {
				fixPrompt, promptErr := r.ai.BuildArchitectureFixPrompt(flags.Output, []string{err.Error()})
				if promptErr != nil {
					return fmt.Errorf("failed to build fix prompt: %w", promptErr)
				}
				if fixErr := r.ai.RunAgent(fixPrompt); fixErr != nil {
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
			fixPrompt, promptErr := r.ai.BuildArchitectureFixPrompt(flags.Output, validationErrors)
			if promptErr != nil {
				return fmt.Errorf("failed to build fix prompt: %w", promptErr)
			}
			if fixErr := r.ai.RunAgent(fixPrompt); fixErr != nil {
				return fmt.Errorf("architecture fix attempt %d failed: %w", attempt, fixErr)
			}
			continue
		}

		return fmt.Errorf("architecture validation failed after %d attempts: %s", maxAttempts, strings.Join(validationErrors, "; "))
	}

	return nil
}

func (r *ArchitectureCmd) commitAndCreatePR(flags ArchitectureFlags) error {
	if !flags.IsWorkflowExecution {
		return nil
	}

	if !r.git.IsFileModifiedOrNew(flags.Output) {
		logger.Infof("No changes detected for %s, skipping PR", flags.Output)
		return nil
	}

	if err := r.git.CheckoutOrCreateBranch("architecture"); err != nil {
		return fmt.Errorf("failed to checkout architecture branch: %w", err)
	}

	if err := r.git.StageFile(flags.Output); err != nil {
		return fmt.Errorf("failed to stage architecture file: %w", err)
	}

	auth := &git.AuthConfig{Owner: flags.RepoOwner, Repo: flags.RepoName}

	if err := r.git.CommitAllAndPush(auth, "architecture", "architecture: generate architecture.yaml"); err != nil {
		return fmt.Errorf("failed to commit and push architecture: %w", err)
	}

	baseBranch := flags.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	proj := &project.Project{
		Slug:  "architecture",
		Title: "Update architecture.yaml",
	}

	prSummary := "Automatically generated architecture.yaml documenting the project structure."

	prURL, err := r.github.CreatePullRequest(proj, "architecture", baseBranch, prSummary)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Successf("Pull request created: %s", prURL)
	return nil
}
