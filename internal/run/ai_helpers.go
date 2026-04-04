package run

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/prompt"
)

// createTempFile creates a temp file under the repo's tmp/ directory so that
// workflow agents, which lack access to /tmp, can read and write it.
func createTempFile(name string) (*os.File, error) {
	path, err := git.TmpPath(name)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
// This matches ralph.sh's approach: agent writes to a file, we read it back
func GeneratePRSummary(ctx *context.Context, proj *project.Project, projectStatus, baseBranch, commitLog string) (summary string, err error) {
	f, err := createTempFile("pr-summary.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary PR summary file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	prPrompt, err := prompt.BuildPRSummaryPrompt(proj.Description, projectStatus, baseBranch, commitLog, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build PR summary prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prPrompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, model, prPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

// runOpenCodeAndReadResult runs opencode with the given prompt and reads the result from the output file
func runOpenCodeAndReadResult(ctx *context.Context, model, prompt, outputFile string) (string, error) {
	var stdoutWriter, stderrWriter io.Writer
	if ctx.IsVerbose() {
		stdoutWriter = os.Stdout
		stderrWriter = os.Stderr
	}

	if err := ai.RunCommand(ctx.GoContext(), model, prompt, stdoutWriter, stderrWriter); err != nil {
		return "", fmt.Errorf("opencode execution failed: %w", err)
	}

	// Read the summary from the file the agent wrote
	summaryBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read summary file: %w", err)
	}

	summary := strings.TrimSpace(string(summaryBytes))
	if summary == "" {
		return "", fmt.Errorf("summary file is empty")
	}

	return summary, nil
}

func resolveModel(ctx *context.Context) string {
	if ctx.Model() != "" {
		return ctx.Model()
	}
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "deepseek/deepseek-chat"
	}
	return ralphConfig.Model
}

// GenerateChangelog prompts opencode to inspect the current git diff and write a
// descriptive changelog to report.md.
func GenerateChangelog(ctx *context.Context) (err error) {
	f, err := createTempFile("changelog.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary changelog file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	changelogPrompt, err := prompt.BuildChangelogPrompt(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to build changelog prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(changelogPrompt)
	}

	model := resolveModel(ctx)
	_, err = runOpenCodeAndReadResult(ctx, model, changelogPrompt, tmpFile)
	if err != nil {
		return err
	}

	// The agent writes to the file we gave it; we need to move that to report.md
	if err = os.Rename(tmpFile, "report.md"); err != nil {
		return fmt.Errorf("failed to rename changelog to report.md: %w", err)
	}

	return nil
}

// GenerateReviewPRBody generates a PR body for review findings using AI
// It reads the review project file and writes a concise summary of recommended changes
func GenerateReviewPRBody(ctx *context.Context, proj *project.Project, requirementSummaries []string) (summary string, err error) {
	f, err := createTempFile("review-pr-body.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary review PR body file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	reviewPrompt, err := prompt.BuildReviewPRBodyPrompt(proj.Name, proj.Description, requirementSummaries, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build review PR body prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(reviewPrompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, model, reviewPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}
