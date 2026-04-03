package run

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/fileutil"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/project"
)

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
// This matches ralph.sh's approach: agent writes to a file, we read it back
func GeneratePRSummary(ctx *context.Context, proj *project.Project, projectStatus, baseBranch, commitLog string) (summary string, err error) {
	tmpFile, err := fileutil.TempFile("pr-summary-", ".md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary PR summary file: %w", err)
	}
	defer fileutil.Remove(tmpFile)

	prompt := buildPRSummaryPrompt(proj.Description, projectStatus, baseBranch, commitLog, tmpFile)

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, model, prompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

type prSummaryData struct {
	ProjectDesc   string
	ProjectStatus string
	BaseBranch    string
	CommitLog     string
	AbsPath       string
}

var prSummaryPromptTemplate = template.Must(template.New("prSummary").Parse(`Write a concise PR description (3-5 paragraphs max) for the changes made in this branch.

Project: {{.ProjectDesc}}
Status: {{.ProjectStatus}}

## Commit Log
{{.CommitLog}}

Review the git commits from {{.BaseBranch}}..HEAD to understand what was changed.
Use 'git log --format="%h: %B" {{.BaseBranch}}..HEAD' to see commit messages.
Use 'git diff {{.BaseBranch}}..HEAD' to see the full changes.

Summarize:
1. What was implemented/changed
2. Key technical decisions
3. Any notable considerations or future work

Be concise and focus on what matters for code review.

Write your summary to the file: {{.AbsPath}}
`))

// buildPRSummaryPrompt constructs the prompt for generating PR summary
func buildPRSummaryPrompt(projectDesc, projectStatus, baseBranch, commitLog, outputFile string) string {
	absPath, _ := fileutil.Abs(outputFile)

	var builder bytes.Buffer
	data := prSummaryData{
		ProjectDesc:   projectDesc,
		ProjectStatus: projectStatus,
		BaseBranch:    baseBranch,
		CommitLog:     commitLog,
		AbsPath:       absPath,
	}
	prSummaryPromptTemplate.Execute(&builder, data)
	return builder.String()
}

// runOpenCodeAndReadResult runs opencode with the given prompt and reads the result from the output file
func runOpenCodeAndReadResult(ctx *context.Context, model, prompt, outputFile string) (string, error) {
	var stdoutWriter, stderrWriter io.Writer
	if ctx.IsVerbose() {
		stdoutWriter = os.Stdout
		stderrWriter = os.Stderr
	}

	if err := opencode.RunCommand(ctx.GoContext(), model, prompt, stdoutWriter, stderrWriter); err != nil {
		return "", fmt.Errorf("opencode execution failed: %w", err)
	}

	// Read the summary from the file the agent wrote
	summaryBytes, err := fileutil.ReadFile(outputFile)
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
	tmpFile, err := fileutil.TempFile("changelog-", ".md")
	if err != nil {
		return fmt.Errorf("failed to create temporary changelog file: %w", err)
	}
	defer fileutil.Remove(tmpFile)

	prompt := buildChangelogPrompt(tmpFile)

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)
	_, err = runOpenCodeAndReadResult(ctx, model, prompt, tmpFile)
	if err != nil {
		return err
	}

	// The agent writes to the file we gave it; we need to move that to report.md
	if err = fileutil.Rename(tmpFile, "report.md"); err != nil {
		return fmt.Errorf("failed to rename changelog to report.md: %w", err)
	}

	return nil
}

var changelogPromptTemplate = template.Must(template.New("changelog").Parse(`Write a concise changelog entry for the changes currently staged in git.

You are an AI agent that writes changelogs. Review the git diff (staged changes) and write a single changelog entry describing what changed.

Focus on:
• What was added, removed, or modified
• Why the changes were made (if apparent from the diff)
• Any notable implementation details

Write in the style of a conventional changelog entry, beginning with a verb in past tense (e.g., "Fixed", "Added", "Changed").

Write the changelog entry to the file: {{.}}

Do not include any extra commentary, just the changelog entry.`))

func buildChangelogPrompt(outputFile string) string {
	absPath, _ := fileutil.Abs(outputFile)
	var b bytes.Buffer
	changelogPromptTemplate.Execute(&b, absPath)
	return b.String()
}

// GenerateReviewPRBody generates a PR body for review findings using AI
// It reads the review project file and writes a concise summary of recommended changes
func GenerateReviewPRBody(ctx *context.Context, proj *project.Project, requirementSummaries []string) (summary string, err error) {
	tmpFile, err := fileutil.TempFile("review-pr-body-", ".md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary review PR body file: %w", err)
	}
	defer fileutil.Remove(tmpFile)

	prompt := buildReviewPRBodyPrompt(proj.Name, proj.Description, requirementSummaries, tmpFile)

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, model, prompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

type reviewPRBodyData struct {
	ProjectName        string
	ProjectDescription string
	Requirements       []string
	AbsPath            string
}

var reviewPRBodyPromptTemplate = template.Must(template.New("reviewPRBody").Parse(`Write a concise PR description (2-4 paragraphs max) for this code review.

Review Name: {{.ProjectName}}
{{if .ProjectDescription}}
Description: {{.ProjectDescription}}
{{end}}

## Findings Summary
{{range .Requirements}}
{{.}}
{{end}}

Review the requirements above and write a summary that:
1. Lists the key findings from the review
2. Highlights any critical issues that need attention
3. Notes what's working well

Write your summary to the file: {{.AbsPath}}
`))

func buildReviewPRBodyPrompt(projectName, projectDesc string, requirements []string, outputFile string) string {
	absPath, _ := fileutil.Abs(outputFile)

	var builder bytes.Buffer
	data := reviewPRBodyData{
		ProjectName:        projectName,
		ProjectDescription: projectDesc,
		Requirements:       requirements,
		AbsPath:            absPath,
	}
	reviewPRBodyPromptTemplate.Execute(&builder, data)
	return builder.String()
}
