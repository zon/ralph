package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/config"
	appcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// opencodeEvent represents a JSON event from OpenCode's --format json output
type opencodeEvent struct {
	Content string `json:"content"`
}

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
// In dry-run mode, it logs what would be executed without actually calling OpenCode
func RunAgent(ctx *appcontext.Context, prompt string) error {
	if ctx.IsDryRun() {
		logger.Info(prompt)
		return nil
	}

	if ctx.IsVerbose() {
		logger.Info(prompt)
	}

	cmd := exec.Command("opencode", "run", prompt)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode execution failed: %w", err)
	}

	return nil
}

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
func GeneratePRSummary(ctx *appcontext.Context, projectFile string, iterations int) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would generate PR summary")
		return "dry-run-pr-summary", nil
	}

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
	commitLog, err := git.GetCommitLog(ctx, baseBranch)
	if err != nil {
		logger.Warningf("Failed to get commit log: %v", err)
		commitLog = "(Unable to retrieve commit log)"
	}

	// Build prompt matching ralph.sh
	var builder strings.Builder
	builder.WriteString("Write a concise PR description (3-5 paragraphs max) for the changes made in this branch.\n\n")
	builder.WriteString(fmt.Sprintf("Project: %s\n", project.Description))
	builder.WriteString(fmt.Sprintf("Status: %s\n\n", projectStatus))
	builder.WriteString("## Commit Log\n")
	builder.WriteString(commitLog)
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("Use 'git diff %s..HEAD' to see the full changes.\n\n", baseBranch))
	builder.WriteString("Summarize:\n")
	builder.WriteString("1. What was implemented/changed\n")
	builder.WriteString("2. Key technical decisions\n")
	builder.WriteString("3. Any notable considerations or future work\n\n")
	builder.WriteString("Be concise and focus on what matters for code review.\n")

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Info(prompt)
	}

	cmd := exec.Command("opencode", "run", "--format", "json", prompt)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("opencode execution failed: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON output to extract content
	// OpenCode outputs one JSON object per line
	lines := strings.Split(string(output), "\n")
	var summary strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event opencodeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip non-JSON lines
		}

		if event.Content != "" {
			summary.WriteString(event.Content)
		}
	}

	result := summary.String()
	if result == "" {
		return "", fmt.Errorf("no content found in opencode output")
	}

	return strings.TrimSpace(result), nil
}
