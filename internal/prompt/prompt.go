package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// BuildDevelopPrompt creates a prompt for the AI agent to work on project requirements
// It includes recent git history, project requirements, and development instructions
func BuildDevelopPrompt(ctx *context.Context, projectFile string) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would build development prompt")
		return "dry-run-prompt", nil
	}

	var builder strings.Builder

	// Header
	builder.WriteString("# Development Agent Context\n\n")
	builder.WriteString("## Project Information\n\n")
	builder.WriteString("You are an AI coding agent working on this project.\n")
	builder.WriteString("Your task is to implement requirements from the project file below.\n\n")

	// Recent Git History
	builder.WriteString("## Recent Git History\n\n")
	commits, err := git.GetRecentCommits(ctx, 20)
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to get recent commits: %v", err))
		builder.WriteString("(Unable to retrieve git history)\n\n")
	} else if len(commits) == 0 {
		builder.WriteString("(No commits yet)\n\n")
	} else {
		for _, commit := range commits {
			builder.WriteString(commit)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Project Requirements
	builder.WriteString("## Project Requirements\n\n")
	projectContent, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}
	builder.Write(projectContent)
	builder.WriteString("\n\n")

	// Development Instructions
	instructionsPath := filepath.Join("docs", "develop-instructions.md")
	if _, err := os.Stat(instructionsPath); err == nil {
		builder.WriteString("## Development Instructions\n\n")
		instructionsContent, err := os.ReadFile(instructionsPath)
		if err != nil {
			logger.Warning(fmt.Sprintf("Failed to read %s: %v", instructionsPath, err))
		} else {
			builder.Write(instructionsContent)
			builder.WriteString("\n")
		}
	} else {
		// Include default instructions if file doesn't exist
		builder.WriteString("## Development Instructions\n\n")
		builder.WriteString(getDefaultInstructions())
		builder.WriteString("\n")
	}

	prompt := builder.String()

	if ctx.IsVerbose() {
		logger.Info(fmt.Sprintf("Generated prompt (%d bytes)", len(prompt)))
	}

	return prompt, nil
}

// getDefaultInstructions returns default development instructions if docs/develop-instructions.md doesn't exist
func getDefaultInstructions() string {
	return `1. Review the requirements in the project file above
2. Look for requirements with 'passing: false' - these need implementation
3. ONLY WORK ON ONE REQUIREMENT. Select which requirement is the highest priority
4. Write tests covering the new functionality BEFORE or ALONGSIDE implementation
   - Tests should verify the requirement's acceptance criteria
   - Run tests to ensure they pass
5. When complete, write a concise report in 'report.md' formatted as a git commit message
   - Include ONLY: brief summary of what was implemented and what tests were added
   - Keep it short and high-level (suitable for a commit message)
   - Do NOT include detailed explanations, code snippets, or implementation details
6. Update the requirement in the project YAML file to 'passing: true' ONLY if:
   - The requirement is fully implemented and complete
   - All tests are passing`
}
