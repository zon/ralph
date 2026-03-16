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

const mockAIEnv = "RALPH_MOCK_AI"

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

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
func RunAgent(ctx *context.Context, prompt string) error {
	if os.Getenv(mockAIEnv) == "true" {
		return runMockAgent(ctx, prompt)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)

	cmd := exec.Command("opencode", "run", "--model", model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	stdoutBuf := &captureWriter{out: os.Stdout}
	stderrBuf := &captureWriter{out: os.Stderr}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if err := cmd.Run(); err != nil {
		tail := combineTails(stdoutBuf, stderrBuf, 10)
		lineCount := len(stdoutBuf.lines) + len(stderrBuf.lines)
		if lineCount > 10 {
			lineCount = 10
		}
		return fmt.Errorf("opencode execution failed: %w\n\nLast %d lines of output:\n%s", err, lineCount, tail)
	}

	return nil
}

type captureWriter struct {
	out   *os.File
	lines []string
}

func (cw *captureWriter) Write(p []byte) (n int, err error) {
	if cw.out != nil {
		cw.out.Write(p)
	}

	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			cw.lines = append(cw.lines, line)
		}
	}
	return len(p), nil
}

func (cw *captureWriter) tail(n int) string {
	if len(cw.lines) == 0 {
		return ""
	}
	start := 0
	if len(cw.lines) > n {
		start = len(cw.lines) - n
	}
	return strings.Join(cw.lines[start:], "\n")
}

func combineTails(stdoutBuf, stderrBuf *captureWriter, n int) string {
	var allLines []string
	allLines = append(allLines, stdoutBuf.lines...)
	allLines = append(allLines, stderrBuf.lines...)
	if len(allLines) == 0 {
		return ""
	}
	start := 0
	if len(allLines) > n {
		start = len(allLines) - n
	}
	return strings.Join(allLines[start:], "\n")
}

// GenerateChangelog prompts opencode to inspect the current git diff and write a
// concise commit-message-style changelog entry to report.md.  It is called when an
// iteration leaves uncommitted changes but the agent did not produce report.md itself.
func GenerateChangelog(ctx *context.Context) error {
	prompt := buildChangelogPrompt()

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)
	cmd := exec.Command("opencode", "run", "--model", model, prompt)
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
func GeneratePRSummary(ctx *context.Context, projectFile string, iterations int, baseBranch string) (string, error) {
	project, err := config.LoadProject(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to load project: %w", err)
	}

	allComplete, _, _ := config.CheckCompletion(project)

	var projectStatus string
	if allComplete {
		projectStatus = "✅ Complete"
	} else {
		projectStatus = "⚠️ Incomplete"
	}

	commitLog, err := git.GetCommitLog(baseBranch, 0)
	if err != nil {
		logger.Verbosef("Failed to get commit log: %v", err)
		commitLog = "(Unable to retrieve commit log)"
	}

	tmpFile, err := createTempSummaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	prompt := buildPRSummaryPrompt(project.Description, projectStatus, baseBranch, commitLog, tmpFile)

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)
	summary, err := runOpenCodeAndReadResult(ctx, model, prompt, tmpFile)
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

// runMockAgent simulates AI execution for testing purposes.
// It parses the prompt to determine what file to write and creates mock output files.
func runMockAgent(ctx *context.Context, prompt string) error {
	if os.Getenv("RALPH_MOCK_AI_FAIL") == "true" {
		return fmt.Errorf("opencode execution failed: mock AI failure\n\nline 9 output\nline 10 output\nline 11 output\nline 12 output")
	}

	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "picked-requirement") {
		absProjectFile := ctx.ProjectFile()
		if absProjectFile == "" {
			return fmt.Errorf("mock AI requires project file to be set")
		}

		pickedReqPath := filepath.Join(filepath.Dir(absProjectFile), "picked-requirement.yaml")
		mockReqContent := `- description: Mock requirement
  passing: false
`
		if err := os.WriteFile(pickedReqPath, []byte(mockReqContent), 0644); err != nil {
			return fmt.Errorf("mock AI failed to write picked-requirement.yaml: %w", err)
		}
		logger.Verbosef("Mock AI wrote picked-requirement.yaml")
	}

	if strings.Contains(promptLower, "report.md") {
		if err := os.WriteFile("report.md", []byte("Mock: test commit\n"), 0644); err != nil {
			return fmt.Errorf("mock AI failed to write report.md: %w", err)
		}
		logger.Verbosef("Mock AI wrote report.md")
	}

	return nil
}
