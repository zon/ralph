package ai

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
// In dry-run mode, it logs what would be executed without actually calling OpenCode
func RunAgent(ctx *context.Context, prompt string) error {
	if ctx.IsDryRun() {
		logger.Verbose(prompt)
		return nil
	}

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
// This matches ralph.sh's approach: agent writes to a file, we read it back
func GeneratePRSummary(ctx *context.Context, projectFile string, iterations int) (string, error) {
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
	commitLog, err := git.GetCommitLog(ctx, baseBranch, 0)
	if err != nil {
		logger.Verbosef("Failed to get commit log: %v", err)
		commitLog = "(Unable to retrieve commit log)"
	}

	// Create temp directory and file for agent to write to
	tmpDir := "tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("pr-summary-%d.txt", os.Getpid()))
	defer os.Remove(tmpFile) // Clean up temp file when done

	// Get absolute path for the temp file
	absPath, err := filepath.Abs(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Build prompt matching ralph.sh - agent writes to file instead of outputting
	var builder strings.Builder
	builder.WriteString("Write a concise PR description (3-5 paragraphs max) for the changes made in this branch.\n\n")
	builder.WriteString(fmt.Sprintf("Project: %s\n", project.Description))
	builder.WriteString(fmt.Sprintf("Status: %s\n\n", projectStatus))
	builder.WriteString("## Commit Log\n")
	builder.WriteString(commitLog)
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("Review the git commits from %s..HEAD to understand what was changed.\n", baseBranch))
	builder.WriteString(fmt.Sprintf("Use 'git log --format=\"%%h: %%B\" %s..HEAD' to see commit messages.\n", baseBranch))
	builder.WriteString(fmt.Sprintf("Use 'git diff %s..HEAD' to see the full changes.\n\n", baseBranch))
	builder.WriteString("Summarize:\n")
	builder.WriteString("1. What was implemented/changed\n")
	builder.WriteString("2. Key technical decisions\n")
	builder.WriteString("3. Any notable considerations or future work\n\n")
	builder.WriteString("Be concise and focus on what matters for code review.\n\n")
	builder.WriteString(fmt.Sprintf("Write your summary to the file: %s\n", absPath))

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	// Run opencode with the configured model, let it write the file
	cmd := exec.Command("opencode", "run", "--model", ralphConfig.Model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	cmd.Stdin = os.Stdin
	if ctx.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("opencode execution failed: %w", err)
	}

	// Read the summary from the file the agent wrote
	summaryBytes, err := os.ReadFile(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to read summary file: %w", err)
	}

	summary := strings.TrimSpace(string(summaryBytes))
	if summary == "" {
		return "", fmt.Errorf("summary file is empty")
	}

	return summary, nil
}
