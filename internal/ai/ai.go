package ai

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
func RunAgent(ctx *context.Context, prompt string) error {
	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	// Load config to get model
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cmd := exec.Command("opencode", "run", "--model", ralphConfig.Model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode execution failed: %w", err)
	}

	return nil
}

// GenerateChangelog prompts opencode to inspect the current git diff and write a
// concise commit-message-style changelog entry to report.md.  It is called when an
// iteration leaves uncommitted changes but the agent did not produce report.md itself.
func GenerateChangelog(ctx *context.Context) error {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	prompt := buildChangelogPrompt()

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	cmd := exec.Command("opencode", "run", "--model", ralphConfig.Model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode changelog generation failed: %w", err)
	}

	return nil
}

var changelogPromptTemplate = template.Must(template.New("changelog").Parse(`Inspect the current uncommitted git changes and write a concise changelog entry to 'report.md'.

Steps:
1. Run 'git diff HEAD' (or 'git diff --cached' for staged-only) to see what changed.
2. Write a short, commit-message-style summary of the changes to 'report.md'.

Format:
- First line: imperative-mood summary (≤72 chars), e.g. "Add user authentication endpoint"
- Optional blank line followed by bullet points for non-obvious details

Keep it concise — this becomes the git commit message.
Do NOT include code snippets or verbose explanations.
`))

// buildChangelogPrompt returns the prompt used to generate report.md from git diff output.
func buildChangelogPrompt() string {
	var b bytes.Buffer
	changelogPromptTemplate.Execute(&b, nil)
	return b.String()
}

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
// This matches ralph.sh's approach: agent writes to a file, we read it back
func GeneratePRSummary(ctx *context.Context, projectFile string, iterations int) (string, error) {
	// Load project file
	project, err := config.LoadProject(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to load project: %w", err)
	}

	// Get completion status
	allComplete, _, _ := config.CheckCompletion(project)

	var projectStatus string
	if allComplete {
		projectStatus = "✅ Complete"
	} else {
		projectStatus = "⚠️ Incomplete"
	}

	// Get base branch
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	baseBranch := ralphConfig.BaseBranch

	// Get commit log since base branch
	commitLog, err := git.GetCommitLog(ctx, baseBranch, 0)
	if err != nil {
		logger.Verbosef("Failed to get commit log: %v", err)
		commitLog = "(Unable to retrieve commit log)"
	}

	// Create temp file for agent to write to
	tmpFile, err := createTempSummaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	// Build prompt and run opencode
	prompt := buildPRSummaryPrompt(project.Description, projectStatus, baseBranch, commitLog, tmpFile)

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	// Run opencode and read result
	summary, err := runOpenCodeAndReadResult(ctx, ralphConfig.Model, prompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

// createTempSummaryFile creates a temporary file for the PR summary
func createTempSummaryFile() (string, error) {
	tmpDir := "tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("pr-summary-%d.txt", os.Getpid()))
	return tmpFile, nil
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
	absPath, _ := filepath.Abs(outputFile)

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
	cmd := exec.Command("opencode", "run", "--model", model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	if ctx.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
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
